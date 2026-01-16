//go:build darwin || linux
// +build darwin linux

package main

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"go.bug.st/serial"
)

var ptyPath string
var ptyPathMu sync.RWMutex

// GetPTYPath returns the current PTY slave path for external applications
func GetPTYPath() string {
	ptyPathMu.RLock()
	defer ptyPathMu.RUnlock()
	return ptyPath
}

// startRigWatcherWithPTY starts the Rig watcher with PTY routing.
// It creates a virtual PTY and routes data between the real COM port and the PTY.
// External applications (WSJT-X, JTDX, etc.) connect to the PTY slave.
func startRigWatcherWithPTY() {
	configLock.RLock()
	use := config.UseRig
	port := config.RigPort
	baud := config.RigBaud
	configLock.RUnlock()

	log.Println("[RIG-PTY] use:", use, "port:", port, "baud:", baud)
	if !use || port == "" {
		log.Println("[RIG-PTY] disabled")
		return
	}
	if baud == 0 {
		baud = 9600
	}

	// Open real COM port
	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	realCOM, err := serial.Open(port, mode)
	if err != nil {
		log.Println("[RIG-PTY] COM open error:", err)
		return
	}
	defer realCOM.Close()
	log.Println("[RIG-PTY] COM opened:", port)

	// グローバルに保存（設定変更時のAI1送信用）
	currentRigPortMu.Lock()
	currentRigPort = realCOM
	currentRigPortMu.Unlock()
	defer func() {
		currentRigPortMu.Lock()
		currentRigPort = nil
		currentRigPortMu.Unlock()
	}()

	// Create PTY pair
	ptmx, tty, err := pty.Open()
	if err != nil {
		log.Println("[RIG-PTY] PTY open error:", err)
		return
	}
	defer ptmx.Close()
	defer tty.Close()

	// Get slave path
	slavePath := tty.Name()

	ptyPathMu.Lock()
	ptyPath = slavePath
	ptyPathMu.Unlock()

	log.Println("[RIG-PTY] PTY created:", slavePath)
	log.Println("[RIG-PTY] Use this path in WSJT-X/JTDX/HAMLOG")

	// Broadcast PTY path to WebSocket clients
	broadcastPTYPath(slavePath)

	var protoMu sync.Mutex
	var proto RigProto
	var catBuf strings.Builder

	// Initial CI-V probe
	go func() {
		time.Sleep(300 * time.Millisecond)
		if proto == ProtoUnknown {
			log.Println("[RIG-PTY] initial poll: CI-V")
			civInitialPoll(realCOM)
		}
		time.Sleep(700 * time.Millisecond)
		protoMu.Lock()
		if proto == ProtoUnknown {
			proto = ProtoCAT
			log.Println("[RIG-PTY] fallback to CAT")
			startCATPollerPTY(realCOM)
		}
		protoMu.Unlock()
	}()

	// PTY書き込み用チャネル（ブロック防止）
	ptyWriteChan := make(chan []byte, 100)

	// Goroutine: PTY書き込みワーカー
	go func() {
		for data := range ptyWriteChan {
			_, err := ptmx.Write(data)
			if err != nil {
				log.Println("[RIG-PTY] PTY write error:", err)
				return
			}
		}
	}()

	// Goroutine: PTY → realCOM (commands from WSJT-X to rig)
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Println("[RIG-PTY] PTY read error:", err)
				}
				return
			}
			if n > 0 {
				data := buf[:n]
				// log.Printf("[RIG-PTY] PTY→COM %d bytes: % X", n, data)
				_, err := realCOM.Write(data)
				if err != nil {
					log.Println("[RIG-PTY] COM write error:", err)
					return
				}
			}
		}
	}()

	// Main loop: realCOM → PTY (responses from rig to WSJT-X)
	buf := make([]byte, 256)
	for {
		n, err := realCOM.Read(buf)
		if err != nil {
			log.Println("[RIG-PTY] COM read error:", err)
			return
		}
		if n == 0 {
			continue
		}

		data := buf[:n]
		// log.Printf("[RIG-PTY] COM→PTY %d bytes: % X", n, data)

		// Forward to PTY (pass-through, non-blocking)
		dataCopy := make([]byte, n)
		copy(dataCopy, data)
		select {
		case ptyWriteChan <- dataCopy:
		default:
			// バッファフル時は破棄（COMの読み取りをブロックしない）
		}

		// Analyze data (mirror mode - same as direct connection)
		protoMu.Lock()
		if proto == ProtoUnknown {
			proto = detectProto(data)
			if proto != ProtoUnknown {
				log.Println("[RIG-PTY] detected protocol:", proto)
				if proto == ProtoCAT {
					startCATPollerPTY(realCOM)
				}
			}
		}
		currentProto := proto
		protoMu.Unlock()

		switch currentProto {
		case ProtoCIV:
			handleCIV(data)

		case ProtoCAT:
			catBuf.Write(data)
			for {
				full := catBuf.String()
				// log.Printf("[RIG-PTY] catBuf: %q", full)
				idx := strings.Index(full, ";")
				if idx < 0 {
					break
				}
				cmd := full[:idx+1]
				rest := full[idx+1:]
				catBuf.Reset()
				catBuf.WriteString(rest)
				handleCATCommandPTY(strings.TrimSuffix(cmd, ";"))
			}
		}
	}
}

// startCATPollerPTY starts CAT polling in PTY mode
func startCATPollerPTY(s serial.Port) {
	// Auto Information ON (YAESUのみ)
	_, _ = s.Write([]byte("AI1;"))
	log.Println("[RIG-PTY] CAT Auto Information enabled")

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			_, _ = s.Write([]byte("FA;MD0;"))
		}
	}()
}

// handleCATCommandPTY handles CAT commands without sending responses
func handleCATCommandPTY(cmd string) {
	if len(cmd) < 2 {
		return
	}
	// log.Println("[RIG-PTY] CAT command:", cmd)

	switch {
	case strings.HasPrefix(cmd, "IF"):
		parseIF(cmd)

	case strings.HasPrefix(cmd, "FA"):
		if freq := parseCATFreq(cmd); freq > 0 {
			updateCATState(freq, "", false)
		}

	case strings.HasPrefix(cmd, "MD"):
		if mode, data := parseCATMode(cmd); mode != "" {
			updateCATState(0, mode, data)
		}
	}
}

// broadcastPTYPath sends the PTY path to WebSocket clients
func broadcastPTYPath(path string) {
	ev := map[string]interface{}{
		"type": "pty",
		"path": path,
	}
	b, _ := json.Marshal(ev)
	broadcast(string(b))
}
