//go:build darwin || linux
// +build darwin linux

package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"go.bug.st/serial"
)

var ptyPaths []string
var ptyPathsMu sync.RWMutex

// PTY再起動制御
var ptyStopFlag bool
var ptyStopMu sync.Mutex

// GetPTYPaths returns the current PTY slave paths for external applications
func GetPTYPaths() []string {
	ptyPathsMu.RLock()
	defer ptyPathsMu.RUnlock()
	result := make([]string, len(ptyPaths))
	copy(result, ptyPaths)
	return result
}

// GetPTYPath returns the first PTY slave path (for backward compatibility)
func GetPTYPath() string {
	ptyPathsMu.RLock()
	defer ptyPathsMu.RUnlock()
	if len(ptyPaths) > 0 {
		return ptyPaths[0]
	}
	return ""
}

// stopRigWatcherWithPTY stops all running PTY watchers
func stopRigWatcherWithPTY() {
	ptyStopMu.Lock()
	ptyStopFlag = true
	ptyStopMu.Unlock()

	// 既存の接続を全てクローズ
	currentRigPortsMu.Lock()
	for _, port := range currentRigPorts {
		if port != nil {
			port.Close()
		}
	}
	currentRigPortsMu.Unlock()

	// 少し待機してgoroutineが終了するのを待つ
	time.Sleep(500 * time.Millisecond)

	ptyStopMu.Lock()
	ptyStopFlag = false
	ptyStopMu.Unlock()

	log.Println("[RIG-PTY] stopped")
}

// restartRigWatcherWithPTY restarts all PTY watchers with new configuration
func restartRigWatcherWithPTY() {
	log.Println("[RIG-PTY] restarting with new configuration...")
	stopRigWatcherWithPTY()
	startRigWatcherWithPTY()
}

// startRigWatcherWithPTY starts the Rig watcher with PTY routing.
// It creates a virtual PTY and routes data between multiple real COM ports and the PTY.
// External applications (WSJT-X, JTDX, etc.) connect to the PTY slave.
// All enabled ports are monitored and forwarded to PTY regardless of broadcast mode.
func startRigWatcherWithPTY() {
	configLock.RLock()
	use := config.UseRig
	rigPorts := make([]RigPortConfig, len(config.RigPorts))
	copy(rigPorts, config.RigPorts)
	configLock.RUnlock()

	if !use {
		log.Println("[RIG-PTY] disabled")
		return
	}

	// 有効なポートをカウント
	var enabledPorts []struct {
		Index int
		Port  string
		Baud  int
	}
	for i, rp := range rigPorts {
		if rp.Port != "" {
			baud := rp.Baud
			if baud == 0 {
				baud = 9600
			}
			enabledPorts = append(enabledPorts, struct {
				Index int
				Port  string
				Baud  int
			}{i, rp.Port, baud})
			log.Printf("[RIG-PTY] port[%d]: %s @ %d baud", i, rp.Port, baud)
		}
	}

	if len(enabledPorts) == 0 {
		log.Println("[RIG-PTY] no ports configured")
		return
	}

	// PTYパスを初期化
	ptyPathsMu.Lock()
	ptyPaths = make([]string, len(rigPorts))
	ptyPathsMu.Unlock()

	var wg sync.WaitGroup

	// 各ポートごとに個別のPTYを作成
	for _, ep := range enabledPorts {
		// Create PTY pair for this port
		ptmx, tty, err := pty.Open()
		if err != nil {
			log.Printf("[RIG-PTY-%d] PTY open error: %v", ep.Index, err)
			continue
		}

		slavePath := tty.Name()

		ptyPathsMu.Lock()
		ptyPaths[ep.Index] = slavePath
		ptyPathsMu.Unlock()

		log.Printf("[RIG-PTY-%d] PTY created: %s", ep.Index, slavePath)

		// COMポートを開く
		mode := &serial.Mode{
			BaudRate: ep.Baud,
			DataBits: 8,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}

		realCOM, err := serial.Open(ep.Port, mode)
		if err != nil {
			log.Printf("[RIG-PTY-%d] COM open error: %v", ep.Index, err)
			ptmx.Close()
			tty.Close()
			continue
		}
		log.Printf("[RIG-PTY-%d] COM opened: %s → PTY: %s", ep.Index, ep.Port, slavePath)

		// グローバルに保存
		currentRigPortsMu.Lock()
		currentRigPorts[ep.Index] = realCOM
		currentRigPortsMu.Unlock()

		// 後方互換性: index 0 の場合は旧変数にもセット
		if ep.Index == 0 {
			currentRigPortMu.Lock()
			currentRigPort = realCOM
			currentRigPortMu.Unlock()
		}

		wg.Add(1)
		go func(index int, port string, com serial.Port, master *os.File, slave *os.File) {
			defer wg.Done()
			defer com.Close()
			defer master.Close()
			defer slave.Close()
			defer func() {
				currentRigPortsMu.Lock()
				delete(currentRigPorts, index)
				currentRigPortsMu.Unlock()
				ptyPathsMu.Lock()
				if index < len(ptyPaths) {
					ptyPaths[index] = ""
				}
				ptyPathsMu.Unlock()
				if index == 0 {
					currentRigPortMu.Lock()
					currentRigPort = nil
					currentRigPortMu.Unlock()
				}
			}()

			runSinglePortPTYIndependent(index, port, com, master)
		}(ep.Index, ep.Port, realCOM, ptmx, tty)
	}

	// Broadcast all PTY paths to WebSocket clients
	broadcastPTYPaths()

	log.Println("[RIG-PTY] Use these paths in WSJT-X/JTDX/HAMLOG")

	wg.Wait()
}

// runSinglePortPTYIndependent handles a single COM port with its own independent PTY
func runSinglePortPTYIndependent(index int, port string, com serial.Port, ptmx *os.File) {
	var protoMu sync.Mutex
	var proto RigProto
	var catBuf strings.Builder

	// PTY書き込み用チャネル（ブロック防止）
	ptyWriteChan := make(chan []byte, 100)

	// Goroutine: PTY書き込みワーカー
	go func() {
		for data := range ptyWriteChan {
			_, err := ptmx.Write(data)
			if err != nil {
				log.Printf("[RIG-PTY-%d] PTY write error: %v", index, err)
				return
			}
		}
	}()

	// Initial CI-V probe
	go func() {
		time.Sleep(300 * time.Millisecond)
		protoMu.Lock()
		currentProto := proto
		protoMu.Unlock()
		if currentProto == ProtoUnknown {
			log.Printf("[RIG-PTY-%d] initial poll: CI-V", index)
			civInitialPoll(com)
		}
		time.Sleep(700 * time.Millisecond)
		protoMu.Lock()
		if proto == ProtoUnknown {
			proto = ProtoCAT
			log.Printf("[RIG-PTY-%d] fallback to CAT", index)
			startCATPollerPTYForPort(index, com)
		}
		protoMu.Unlock()
	}()

	// Goroutine: PTY → COM (commands from external app to this rig)
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("[RIG-PTY-%d] PTY read error: %v", index, err)
				}
				return
			}
			if n > 0 {
				_, err := com.Write(buf[:n])
				if err != nil {
					log.Printf("[RIG-PTY-%d] COM write error: %v", index, err)
					return
				}
			}
		}
	}()

	// Main loop: COM → PTY (responses from rig to external app)
	buf := make([]byte, 256)
	for {
		// 停止フラグチェック
		ptyStopMu.Lock()
		if ptyStopFlag {
			ptyStopMu.Unlock()
			log.Printf("[RIG-PTY-%d] stopping (restart requested)", index)
			return
		}
		ptyStopMu.Unlock()

		n, err := com.Read(buf)
		if err != nil {
			log.Printf("[RIG-PTY-%d] COM read error: %v", index, err)
			return
		}
		if n == 0 {
			continue
		}

		data := buf[:n]

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
				log.Printf("[RIG-PTY-%d] detected protocol: %s", index, proto)
				if proto == ProtoCAT {
					startCATPollerPTYForPort(index, com)
				}
			}
		}
		currentProto := proto
		protoMu.Unlock()

		switch currentProto {
		case ProtoCIV:
			handleCIVForPort(index, data)

		case ProtoCAT:
			catBuf.Write(data)
			for {
				full := catBuf.String()
				idx := strings.Index(full, ";")
				if idx < 0 {
					break
				}
				cmd := full[:idx+1]
				rest := full[idx+1:]
				catBuf.Reset()
				catBuf.WriteString(rest)
				handleCATCommandPTY(index, strings.TrimSuffix(cmd, ";"))
			}
		}
	}
}

// startCATPollerPTYForPort starts CAT polling in PTY mode for a specific port.
// If the rig doesn't respond within timeout, it falls back to polling mode
// for older rigs that don't support AI1 (e.g., FT-817, FT-857, TS-2000).
func startCATPollerPTYForPort(index int, s serial.Port) {
	// CATプロトコル=YAESU/KENWOODなのでAI1を送信
	_, _ = s.Write([]byte("AI1;FA;MD0;"))
	log.Printf("[RIG-PTY-%d] CAT Auto Information enabled", index)

	// Wait and check if AI1 is working
	go func() {
		time.Sleep(2 * time.Second)

		// Check if we received any data
		rigStatesMu.RLock()
		state := rigStates[index]
		hasData := state != nil && state.Freq > 0
		rigStatesMu.RUnlock()

		if !hasData {
			// AI1 not working, start polling mode
			log.Printf("[RIG-PTY-%d] AI1 not responding, starting polling mode (legacy rig support)", index)
			startCATPollingLoopPTY(index, s)
		}
	}()
}

// startCATPollingLoopPTY starts a polling loop for older rigs in PTY mode.
func startCATPollingLoopPTY(index int, s serial.Port) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check if port is still valid
		currentRigPortsMu.Lock()
		_, exists := currentRigPorts[index]
		currentRigPortsMu.Unlock()
		if !exists {
			log.Printf("[RIG-PTY-%d] polling stopped (port closed)", index)
			return
		}

		_, _ = s.Write([]byte("FA;MD0;"))
	}
}

// handleCATCommandPTY handles CAT commands for PTY mode
func handleCATCommandPTY(index int, cmd string) {
	if len(cmd) < 2 {
		return
	}

	if !shouldBroadcastFromPort(index) {
		return
	}

	switch {
	case strings.HasPrefix(cmd, "IF"):
		parseIFForPortPTY(index, cmd)

	case strings.HasPrefix(cmd, "FA"):
		if freq := parseCATFreq(cmd); freq > 0 {
			updateRigStateForPort(index, freq, "", false, ProtoCAT)
		}

	case strings.HasPrefix(cmd, "MD"):
		if mode, data := parseCATMode(cmd); mode != "" {
			updateRigStateForPort(index, 0, mode, data, ProtoCAT)
		}
	}
}

// parseIFForPortPTY parses IF command for PTY mode
func parseIFForPortPTY(index int, cmd string) {
	if len(cmd) < 30 {
		return
	}

	// 周波数
	freqStr := cmd[5:16] // P2
	var hz int64
	for _, c := range freqStr {
		if c < '0' || c > '9' {
			return
		}
		hz = hz*10 + int64(c-'0')
	}

	// モード
	modeCode := cmd[26:27] // P6
	mode, data := parseCATMode("MD0" + modeCode)

	updateRigStateForPort(index, hz, mode, data, ProtoCAT)
}

// broadcastPTYPaths sends the PTY paths to WebSocket clients
func broadcastPTYPaths() {
	ptyPathsMu.RLock()
	paths := make([]string, len(ptyPaths))
	copy(paths, ptyPaths)
	ptyPathsMu.RUnlock()

	ev := map[string]interface{}{
		"type":  "pty",
		"paths": paths,
	}
	b, _ := json.Marshal(ev)
	broadcast(string(b))
}
