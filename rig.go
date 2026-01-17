package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

type RigProto string

const (
	ProtoUnknown RigProto = ""
	ProtoCIV     RigProto = "ICOM"
	ProtoCAT     RigProto = "CAT" // YAESU / KENWOOD
)

type RigMode string

const (
	ModeLSB   RigMode = "LSB"
	ModeUSB   RigMode = "USB"
	ModeCW    RigMode = "CW"
	ModeCWR   RigMode = "CW-R"
	ModeAM    RigMode = "AM"
	ModeFM    RigMode = "FM"
	ModeRTTY  RigMode = "RTTY"
	ModeRTTYR RigMode = "RTTY-R"
	ModeDV    RigMode = "DV"
	ModeWFM   RigMode = "WFM"
)

type RigState struct {
	Freq  int64
	Mode  RigMode // USB / LSB / FM / CW / AM
	Data  bool    // DATA ON / OFF
	Proto RigProto
	Index int // ポートインデックス
}

var rigState = &RigState{}
var rigMu sync.Mutex

// 複数ポート対応
var rigStates = make(map[int]*RigState)
var rigStatesMu sync.RWMutex

// 最後にアクティブだったポートを追跡
var lastActivePort int = -1
var lastActivePortMu sync.Mutex
var lastActiveTime time.Time // 最後にアクティブになった時刻

var currentRigPort serial.Port
var currentRigPortMu sync.Mutex

// 複数ポート対応
var currentRigPorts = make(map[int]serial.Port)
var currentRigPortsMu sync.Mutex

var pendingMode string
var pendingData bool

// ---- public entry ----

// startRigWatcher starts the Rig watcher for all configured ports.
// It reads the Rig configuration from the Config struct and starts
// goroutines for each enabled port.
func startRigWatcher() {
	configLock.RLock()
	use := config.UseRig
	usePTY := config.UsePTY
	rigPorts := make([]RigPortConfig, len(config.RigPorts))
	copy(rigPorts, config.RigPorts)
	broadcastMode := config.RigBroadcastMode
	selectedIndex := config.SelectedRigIndex
	configLock.RUnlock()

	// PTYモードの場合は別関数へ（macOS/Linuxのみ）
	if usePTY && (runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		startRigWatcherWithPTY()
		return
	}

	if !use {
		log.Println("[RIG] disabled")
		return
	}

	// 有効なポートをカウント
	enabledCount := 0
	for i, rp := range rigPorts {
		if rp.Port != "" {
			log.Printf("[RIG] port[%d]: %s @ %d baud", i, rp.Port, rp.Baud)
			enabledCount++
		}
	}

	if enabledCount == 0 {
		log.Println("[RIG] no ports configured")
		return
	}

	log.Printf("[RIG] broadcast mode: %s, selected index: %d", broadcastMode, selectedIndex)

	// 各ポートの監視を開始
	for i, rp := range rigPorts {
		if rp.Port == "" {
			continue
		}
		go startSingleRigWatcher(i, rp.Port, rp.Baud)
	}
}

// startSingleRigWatcher starts a Rig watcher for a single port.
func startSingleRigWatcher(index int, port string, baud int) {
	if baud == 0 {
		baud = 9600
	}

	log.Printf("[RIG-%d] open: %s @ %d baud", index, port, baud)

	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	s, err := serial.Open(port, mode)
	if err != nil {
		log.Printf("[RIG-%d] open error: %v", index, err)
		return
	}
	defer s.Close()

	// グローバルに保存（設定変更時のAI1送信用）
	currentRigPortsMu.Lock()
	currentRigPorts[index] = s
	currentRigPortsMu.Unlock()
	defer func() {
		currentRigPortsMu.Lock()
		delete(currentRigPorts, index)
		currentRigPortsMu.Unlock()
	}()

	// 後方互換性: index 0 の場合は旧変数にもセット
	if index == 0 {
		currentRigPortMu.Lock()
		currentRigPort = s
		currentRigPortMu.Unlock()
		defer func() {
			currentRigPortMu.Lock()
			currentRigPort = nil
			currentRigPortMu.Unlock()
		}()
	}

	// ポートごとの状態を初期化
	rigStatesMu.Lock()
	rigStates[index] = &RigState{Index: index}
	rigStatesMu.Unlock()

	buf := make([]byte, 256)

	var protoMu sync.Mutex
	var proto RigProto

	var catBuf strings.Builder
	var civBuf []byte    // CI-V用バッファ（分割受信対策）
	var detectBuf []byte // プロトコル検出用バッファ（CI-V分割受信対策）

	// --- 初期CI-V探査 ---
	go func() {
		time.Sleep(300 * time.Millisecond)
		protoMu.Lock()
		currentProto := proto
		protoMu.Unlock()
		if currentProto == ProtoUnknown {
			log.Printf("[RIG-%d] initial poll: CI-V", index)
			civInitialPoll(s)
		}
		time.Sleep(700 * time.Millisecond)
		protoMu.Lock()
		if proto == ProtoUnknown {
			proto = ProtoCAT
			log.Printf("[RIG-%d] fallback to CAT", index)
			startCATPoller(index, s)
		}
		protoMu.Unlock()
	}()

	for {
		n, err := s.Read(buf)
		if err != nil {
			log.Printf("[RIG-%d] read error: %v", index, err)
			return
		}
		if n == 0 {
			continue
		}

		data := buf[:n]

		protoMu.Lock()
		if proto == ProtoUnknown {
			// プロトコル未確定時はバッファに蓄積して判定
			detectBuf = append(detectBuf, data...)
			proto = detectProto(detectBuf)
			if proto != ProtoUnknown {
				log.Printf("[RIG-%d] detected protocol: %s", index, proto)
				if proto == ProtoCAT {
					startCATPoller(index, s)
				}
				// 検出成功後、蓄積データを処理対象にする
				data = detectBuf
				detectBuf = nil
			} else {
				// まだ検出できない場合、バッファが大きくなりすぎないよう制限
				if len(detectBuf) > 64 {
					detectBuf = detectBuf[len(detectBuf)-32:]
				}
				protoMu.Unlock()
				continue // 次の読み取りを待つ
			}
		}
		currentProto := proto
		protoMu.Unlock()

		switch currentProto {
		case ProtoCIV:
			if !shouldBroadcastFromPort(index) {
				break
			}
			// CI-Vバッファに追加
			civBuf = append(civBuf, data...)
			// 完全なフレームを処理
			for {
				start := bytes.Index(civBuf, []byte{0xFE, 0xFE})
				if start < 0 {
					// FE FE がない場合、単独FEがあれば残す
					lastFE := bytes.LastIndexByte(civBuf, 0xFE)
					if lastFE >= 0 {
						civBuf = civBuf[lastFE:] // 最後のFE以降を残す
					} else {
						civBuf = nil
					}
					break
				}
				if start > 0 {
					civBuf = civBuf[start:] // FE FE の前のゴミを除去
				}
				end := bytes.IndexByte(civBuf[2:], 0xFD) // FE FE の後から FD を探す
				if end < 0 {
					break // FD がまだ来ていない
				}
				frame := civBuf[:2+end+1]
				parseCIVFrameForPort(index, frame)
				civBuf = civBuf[2+end+1:]
			}

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

				handleCATCommandForPort(index, strings.TrimSuffix(cmd, ";"), s)
			}
		}
	}
}

// detectProto determines the protocol of the given byte slice.
// It returns ProtoUnknown if the protocol cannot be determined.
// Currently, it supports CI-V and CAT protocols.
// CI-V protocol is detected by the presence of 0xFE 0xFE at the start of the frame.
// CAT protocol is detected by the presence of a semicolon (:) in the frame.
// detectProto determines the protocol of the given byte slice.
func detectProto(b []byte) RigProto {
	// CI-V: FE FE パターンを検索（連続していなくても最初のFEがあればCI-Vの可能性）
	if bytes.Contains(b, []byte{0xFE, 0xFE}) {
		return ProtoCIV
	}

	// 単独の FE が含まれている場合はまだ判定しない（次のデータを待つ）
	if bytes.Contains(b, []byte{0xFE}) {
		return ProtoUnknown // まだ判定しない
	}

	s := string(b)
	if strings.Contains(s, "FA") ||
		strings.Contains(s, "MD") {
		return ProtoCAT
	}
	return ProtoUnknown
}

// civInitialPoll sends an initial poll command to the CI-V device.
// It sends two commands: one to get the current frequency and one to get the current mode.
// The commands are sent in the format of FE FE [to] [from] CMD FD, where to/from is 0x00 (broadcast).
// If the current frequency or mode is not known, the corresponding command is sent.
func civInitialPoll(s serial.Port) {
	// FE FE [to] [from] CMD FD
	// to/from は 0x00（ブロードキャスト）でOK
	if rigState.Freq == 0 {
		s.Write([]byte{0xFE, 0xFE, 0x00, 0x00, 0x03, 0xFD}) // freq
	}
	if rigState.Mode == "" {
		s.Write([]byte{0xFE, 0xFE, 0x00, 0x00, 0x04, 0xFD}) // mode
	}
}

// handleCIV parses the given byte slice as a CI-V frame.
// It extracts the individual frames from the byte slice, and
// passes them to parseCIVFrame for further processing.
// The function will continue to parse the byte slice until no
// more frames can be found.
// The CI-V frame is defined as follows: FE FE to cmd ... FD.
// The cmd is a single byte that represents the command.
// The frame is terminated by a single FD byte.
// The function will skip any invalid frames.
func handleCIV(b []byte) {
	// CI-V frame: FE FE to from cmd ... FD
	for {
		start := bytes.Index(b, []byte{0xFE, 0xFE})
		if start < 0 {
			return
		}
		end := bytes.IndexByte(b[start:], 0xFD)
		if end < 0 {
			return
		}
		frame := b[start : start+end+1]
		parseCIVFrame(frame)
		b = b[start+end+1:]
	}
}

// handleCIVForPort handles CI-V data for a specific port index
func handleCIVForPort(index int, b []byte) {
	if !shouldBroadcastFromPort(index) {
		return
	}
	for {
		start := bytes.Index(b, []byte{0xFE, 0xFE})
		if start < 0 {
			return
		}
		end := bytes.IndexByte(b[start:], 0xFD)
		if end < 0 {
			return
		}
		frame := b[start : start+end+1]
		parseCIVFrameForPort(index, frame)
		b = b[start+end+1:]
	}
}

// parseCIVFrameForPort parses CI-V frame for a specific port
func parseCIVFrameForPort(index int, f []byte) {
	if len(f) < 7 {
		return
	}

	cmd := f[4]

	switch cmd {
	case 0x00, 0x03:
		if freq := parseCIVFreq(f); freq > 0 {
			updateRigStateForPort(index, freq, "", false, ProtoCIV)
		}
	case 0x01, 0x04:
		if mode, data := parseCIVMode(f); mode != "" {
			updateRigStateForPort(index, 0, string(mode), data, ProtoCIV)
		}
	}
}

// parseCIVFrame parses the given byte slice as a CI-V frame.
// It extracts the individual frames from the byte slice, and
// passes them to parseCIVFreq or parseCIVMode for further processing.
// The function will continue to parse the byte slice until no
// more frames can be found.
// The CI-V frame is defined as follows: FE FE to cmd ... FD.
// The cmd is a single byte that represents the command.
// The frame is terminated by a single FD byte.
// The function will skip any invalid frames.
func parseCIVFrame(f []byte) {
	if len(f) < 7 {
		return
	}

	cmd := f[4]

	switch cmd {
	case 0x00, 0x03:
		if freq := parseCIVFreq(f); freq > 0 {
			//broadcastRig("ICOM", freq, "", false)
			rigMu.Lock()
			rigState.Freq = freq
			rigState.Proto = ProtoCIV
			rigMu.Unlock()
			broadcastRigState()
		}
	case 0x01, 0x04:
		if mode, data := parseCIVMode(f); mode != "" {
			//broadcastRig("ICOM", 0, string(mode), data)
			rigMu.Lock()
			rigState.Mode = mode
			rigState.Data = data
			rigState.Proto = ProtoCIV
			rigMu.Unlock()
			broadcastRigState()

		}
	}
}

// parseCIVFreq parses the given byte slice as a CI-V frequency frame.
// It expects the frame to be in the format of BCD, little endian.
// The function will return 0 if the frame is invalid.
// The function will return the parsed frequency in Hz otherwise.
func parseCIVFreq(f []byte) int64 {
	// FE FE to from CMD [BCD...] FD
	if len(f) < 9 {
		return 0
	}

	end := len(f) - 1 // FD
	data := f[5:end]  // CMD の次から FD 手前

	var hz int64
	mul := int64(1)

	for i := 0; i < len(data); i++ {
		lo := int64(data[i] & 0x0F)
		hi := int64((data[i] >> 4) & 0x0F)

		hz += lo * mul
		mul *= 10
		hz += hi * mul
		mul *= 10
	}

	return hz
}

/*
Parse CI-V mode parses the given byte slice as a CI-V mode frame.
It expects the frame to be in the format of BCD, little endian.
The function will return 0 if the frame is invalid.
The function will return the parsed frequency in Hz otherwise.
The function takes a byte slice as input and returns two values:
  - A string representing the mode (e.g. "ModeLSB", "modeUSB", etc etc "modeAM", "modeCW", "modeRTTY", "modeRTYR", "modeRTYR", "modeFM", "modeWFM", "modeDV")
  - A boolean representing whether the frame is valid data (true) or invalid data (false)

The function will return 0 if the frame is invalid.
The function will return the parsed frequency in Hz otherwise.
The function takes a byte slice as input and returns two values:
  - A string representing the mode (e.g. "modeLSB", "modeUSB",  "modeAM", "modeCW", "modeRTY", "modeRTYR", "modeRTYR", "modeFM", "modeWFM", "modeDV")
  - A boolean representing whether the frame is valid data (true) or invalid data (false)
*/
func parseCIVMode(f []byte) (mode RigMode, data bool) {
	if len(f) < 7 {
		return "", false
	}

	modeByte := f[5]
	dataByte := f[6]

	switch modeByte {

	// SSB
	case 0x00:
		mode = ModeLSB
	case 0x01:
		mode = ModeUSB

	// 基本
	case 0x02:
		mode = ModeAM
	case 0x03:
		mode = ModeCW
	case 0x07:
		mode = ModeCWR

	// RTTY 系（IC-705 / 旧機種 両対応）
	case 0x04:
		mode = ModeRTTY // IC-705
	case 0x08:
		mode = ModeRTTYR // IC-705
	case 0x0A:
		mode = ModeRTTYR // 旧機種

	// FM 系
	case 0x05:
		mode = ModeFM
	case 0x06:
		mode = ModeWFM

	// デジタル
	case 0x17:
		mode = ModeDV

	default:
		return "", false
	}
	// DATA ON/OFF（機種差を吸収）
	if dataByte == 0x01 {
		data = true
	}

	return
}

// isDStarDR checks if the given frequency is within the D-STAR DR mode frequency range.
// The function returns true if the frequency is within the range, and false otherwise.
// The supported frequency ranges are as follows:
// - 430MHz帯: 434.000.000 - 435.000.000, 439.000.000 - 440.000.000
// - 1200MHz帯: 1.270.000.000 - 1.273.000.000, 1.290.000.000 - 1.293.000.000, 1.299.000.000 - 1.300.000.000
func isDStarDR(freq int64) bool {
	// 430MHz帯
	if freq >= 434_000_000 && freq < 435_000_000 {
		return true
	}
	if freq >= 439_000_000 && freq < 440_000_000 {
		return true
	}

	// 1200MHz帯
	if freq >= 1_270_000_000 && freq < 1_273_000_000 {
		return true
	}
	if freq >= 1_290_000_000 && freq < 1_293_000_000 {
		return true
	}
	if freq >= 1_299_000_000 && freq <= 1_300_000_000 {
		return true
	}

	return false
}

// updateCATState updates the current Rig state with the given frequency,
// mode, and data. If the frequency is greater than 0, it updates
// the frequency of the Rig state. If the mode is not empty, it
// updates the mode of the Rig state. After updating the Rig state,
// it calls broadcastRigState to broadcast the updated Rig state
// to all connected WebSocket clients.
func updateCATState(freq int64, mode string, data bool) {
	rigMu.Lock()
	if freq > 0 {
		rigState.Freq = freq
	}
	if mode != "" {
		rigState.Mode = RigMode(mode)
		rigState.Data = data
	}
	rigState.Proto = ProtoCAT
	rigMu.Unlock()

	broadcastRigState()
}

// shouldBroadcastFromPort checks if data from the given port should be broadcast
// based on the current broadcast mode and selected port index.
func shouldBroadcastFromPort(index int) bool {
	configLock.RLock()
	mode := config.RigBroadcastMode
	selectedIndex := config.SelectedRigIndex
	configLock.RUnlock()

	//log.Printf("[RIG] shouldBroadcast: index=%d, mode=%q, selectedIndex=%d", index, mode, selectedIndex)

	if mode == "single" {
		return index == selectedIndex
	}
	// "all" mode: always broadcast
	return true
}

// updateRigStateForPort updates the rig state for a specific port
func updateRigStateForPort(index int, freq int64, mode string, data bool, proto RigProto) {
	// Update port-specific state
	rigStatesMu.Lock()
	if rigStates[index] == nil {
		rigStates[index] = &RigState{Index: index}
	}

	// 変化があるかチェック
	freqChanged := false
	modeChanged := false

	// 周波数の変化検出（100Hz以上の変化のみ）
	if freq > 0 {
		diff := freq - rigStates[index].Freq
		if diff < 0 {
			diff = -diff
		}
		if diff >= 100 {
			rigStates[index].Freq = freq
			freqChanged = true
		}
	}

	// モードの変化検出
	if mode != "" && (rigStates[index].Mode != RigMode(mode) || rigStates[index].Data != data) {
		rigStates[index].Mode = RigMode(mode)
		rigStates[index].Data = data
		modeChanged = true
	}
	rigStates[index].Proto = proto
	portState := *rigStates[index]
	rigStatesMu.Unlock()

	// 変化がなければブロードキャストしない
	if !freqChanged && !modeChanged {
		return
	}

	// アクティブポートのチェック
	// 別のポートが最近500ms以内にアクティブだった場合、周波数のみの変化は無視
	lastActivePortMu.Lock()
	now := time.Now()
	if lastActivePort != -1 && lastActivePort != index {
		if now.Sub(lastActiveTime) < 500*time.Millisecond {
			// 別のポートがアクティブで、500ms以内
			if !modeChanged {
				// モード変更がない場合は無視（CI-Vトランシーブのノイズ）
				lastActivePortMu.Unlock()
				return
			}
		}
	}
	// アクティブポートを更新
	lastActivePort = index
	lastActiveTime = now
	lastActivePortMu.Unlock()

	// Update global state
	rigMu.Lock()
	if freqChanged {
		rigState.Freq = portState.Freq
	}
	if modeChanged {
		rigState.Mode = portState.Mode
		rigState.Data = portState.Data
	} else if freqChanged && portState.Mode != "" {
		// 周波数のみ変化でポートのモードが設定済みの場合
		rigState.Mode = portState.Mode
		rigState.Data = portState.Data
	}
	// モードが未設定の場合はグローバルのモードを維持
	rigState.Proto = proto
	rigState.Index = index
	rigMu.Unlock()

	broadcastRigState()
}

// handleCATCommandForPort handles a CAT command for a specific port
func handleCATCommandForPort(index int, cmd string, s serial.Port) {
	if len(cmd) < 2 {
		return
	}

	if !shouldBroadcastFromPort(index) {
		return
	}

	switch {
	case strings.HasPrefix(cmd, "IF"):
		parseIFForPort(index, cmd)
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

// parseIFForPort parses IF command for a specific port
func parseIFForPort(index int, cmd string) {
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

var lastFreqPoll time.Time
var lastMDPoll time.Time

// startCATPoller enables CAT Auto Information mode.
// If the rig doesn't respond within timeout, it falls back to polling mode
// for older rigs that don't support AI1 (e.g., FT-817, FT-857, TS-2000).
func startCATPoller(index int, s serial.Port) {
	// Auto Information ON (YAESU/KENWOOD)
	_, _ = s.Write([]byte("AI1;FA;MD0;"))
	log.Printf("[RIG-%d] CAT Auto Information enabled", index)

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
			log.Printf("[RIG-%d] AI1 not responding, starting polling mode (legacy rig support)", index)
			startCATPollingLoop(index, s)
		}
	}()
}

// startCATPollingLoop starts a polling loop for older rigs that don't support AI1.
// This enables support for legacy rigs like FT-817, FT-857, FT-897, TS-2000, etc.
func startCATPollingLoop(index int, s serial.Port) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check if port is still valid
		currentRigPortsMu.Lock()
		_, exists := currentRigPorts[index]
		currentRigPortsMu.Unlock()
		if !exists {
			log.Printf("[RIG-%d] polling stopped (port closed)", index)
			return
		}

		_, _ = s.Write([]byte("FA;MD0;"))
	}
}

// handleCATCommand handles a CAT command received from the serial port.
// It expects the command to be one of the following:
// - "IF" followed by a 9-digit frequency in Hz
// - "FA" followed by a 9-digit frequency in Hz
// - "MD" followed by a single digit mode code
// The function will return without doing anything if the command is invalid.
// The function will parse the command and call updateCATState with the parsed frequency and mode.
// If the command is "IF", the function will call updateCATState with the parsed frequency and an empty string for the mode.
// If the command is "FA", the function will call updateCATState with the parsed frequency and an empty string for the mode.
// If the command is "MD", the function will call updateCATState with 0 for the frequency and the parsed mode and data.
// The function will log a message at the INFO level with the command received.
// The function takes two parameters: the command as a string and the serial port as a serial.Port.
func handleCATCommand(cmd string, s serial.Port) {
	if len(cmd) < 2 {
		return
	}
	//log.Println("[RIG] CAT command:", cmd)

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

// parseIF parses the given string as a CAT IF frame.
// It expects the frame to be in the format of "IF P1(x) P2(x) P3(x) P4(x) P5(x) P6(x) P7(x) P8(x) P9(x) P10(x)".
// The function will return 0 if the frame is invalid.
// The function will return the parsed frequency in Hz otherwise.
func parseIF(cmd string) {
	// IF P1(3) P2(11) P3(5) P4(1) P5(1) P6(1) P7(1) P8(1) P9(2) P10(1)
	// IF004430698000+000000E00000
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

	updateCATState(hz, mode, data)
}

// parseCATFreq parses the given string as a CAT frequency frame.
// It expects the frame to be in the format of "FA" followed by a 9-digit frequency in Hz.
// The function will return 0 if the frame is invalid.
// The function will return the parsed frequency in Hz otherwise.
func parseCATFreq(s string) int64 {
	// FA00014074000
	if len(s) < 5 {
		return 0
	}
	var hz int64
	_ = binary.Read(
		strings.NewReader(s[2:]),
		binary.BigEndian,
		&hz,
	)
	// 上記は簡易。実際は atoi が安全
	hz = 0
	for _, c := range s[2:] {
		if c < '0' || c > '9' {
			break
		}
		hz = hz*10 + int64(c-'0')
	}
	return hz
}

// parseCATMode parses the given string as a CAT mode frame.
// It expects the frame to be in the format of "MD" followed by a single digit mode code.
// The function will return an empty string if the frame is invalid.
// The function will return the parsed mode as a string otherwise.
// The supported modes are as follows:
// - 0: FM (Frequency Modulation)
// - 1: LSB (Lower Side Band)
// - 2: USB (Upper Side Band)
// - 3: CW (Continuous Wave)
// - 4: AM (Amplitude Modulation)
// - 5: RTTY (Radio Teletype)
// - 8: USB (Upper Side Band) with data
// - 18: FM (Frequency Modulation) with data
// The function will return the parsed mode as a string and a boolean indicating whether the mode has data.
func parseCATMode(s string) (string, bool) {
	if len(s) < 4 {
		return "", false
	}

	// MD02 -> "02"
	code := strings.TrimLeft(s[2:], "0")
	data := false

	switch code {
	case "1":
		return "LSB", data
	case "2":
		return "USB", data
	case "3":
		return "CW-U", data
	case "4":
		return "FM", data
	case "5":
		return "AM", data
	case "6":
		return "RTTY-LSB", data
	case "7":
		return "CW-R", data
	case "8":
		data = true
		return "LSB", data
	case "9":
		data = true
		return "RTTY-USB", data
	case "A":
		data = true
		return "FM", data
	case "B":
		return "FM-N", data
	case "C":
		data = true
		return "USB", data
	case "D":
		return "AM-N", data
	case "E":
		return "C4FM", data
	default:
		return "", false
	}
}

// BroadcastRIG to all connected WebSocket clients.
func broadcastRig(rig string, freq int64, mode string, data bool) {
	ev := map[string]interface{}{
		"type": "rig",
		"rig":  rig,
	}
	if freq > 0 {
		ev["freq"] = freq
	}
	if mode != "" {
		ev["mode"] = mode
		ev["data"] = data
	}

	b, _ := json.Marshal(ev)
	broadcast(string(b))
}

// SendAI1 sends AI1; command to enable Auto Information on YAESU rigs.
// Called from webui when settings are changed.
func SendAI1() {
	rigMu.Lock()
	proto := rigState.Proto
	rigMu.Unlock()

	if proto != ProtoCAT {
		return
	}

	currentRigPortMu.Lock()
	defer currentRigPortMu.Unlock()

	if currentRigPort != nil {
		_, _ = currentRigPort.Write([]byte("AI1;FA;MD0;"))
		log.Println("[RIG] AI1; sent (settings changed)")
	}
}

// Broadcast the current rig state to all connected WebSocket clients.
// The rig state is marshaled into a JSON object and sent as a WebSocket message.
// The JSON object contains the following fields:
// - type: a string indicating the type of the message ("rig")
// - rig: the current rig state as a RigState proto
// - freq: the current frequency in Hz (if greater than 0)
// - mode: the current mode as a string (if not empty)
// - data: a boolean indicating whether the mode is valid data (if not empty)
var lastBroadcast struct {
	Freq  int64
	Mode  RigMode
	Data  bool
	Proto RigProto
}

func broadcastRigState() {
	rigMu.Lock()
	defer rigMu.Unlock()

	// 前回と同じ内容ならスキップ
	if rigState.Freq == lastBroadcast.Freq &&
		rigState.Mode == lastBroadcast.Mode &&
		rigState.Data == lastBroadcast.Data &&
		rigState.Proto == lastBroadcast.Proto {
		return
	}

	// 今回の状態を保存
	lastBroadcast.Freq = rigState.Freq
	lastBroadcast.Mode = rigState.Mode
	lastBroadcast.Data = rigState.Data
	lastBroadcast.Proto = rigState.Proto

	ev := map[string]interface{}{
		"type": "rig",
		"rig":  rigState.Proto,
	}

	if rigState.Freq > 0 {
		ev["freq"] = rigState.Freq
	}

	if rigState.Mode != "" {
		mode := rigState.Mode

		// D-STAR 判定
		if mode == ModeDV && rigState.Freq > 0 {
			if isDStarDR(rigState.Freq) {
				ev["mode"] = "D-STAR (DR)"
			} else {
				ev["mode"] = "D-STAR (DV)"
			}
		} else {
			ev["mode"] = mode
		}

		ev["data"] = rigState.Data
	}

	b, _ := json.Marshal(ev)
	broadcast(string(b))
}
