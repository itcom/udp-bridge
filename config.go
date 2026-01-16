package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type RigPortConfig struct {
	Port string `json:"port"`
	Baud int    `json:"baud"`
}

type Config struct {
	QRZUser string `json:"qrz_user"`
	QRZPass string `json:"qrz_pass"`

	UseQRZ bool `json:"use_qrz"`
	UseGeo bool `json:"use_geo"`

	UseRig  bool   `json:"use_rig"`
	RigPort string `json:"rig_port"`
	RigBaud int    `json:"rig_baud"`
	UsePTY  bool   `json:"use_pty"`

	// 複数ポート対応
	RigPorts         []RigPortConfig `json:"rig_ports"`
	RigBroadcastMode string          `json:"rig_broadcast_mode"` // "single" or "all"
	SelectedRigIndex int             `json:"selected_rig_index"` // "single"モード時のインデックス
}

var (
	config     Config
	configLock sync.RWMutex
)

// loadConfig loads the configuration from a file named "config.json".
// If the file does not exist, the configuration is left unchanged.
// If the file exists, it is unmarshaled into JSON and the configuration is updated.
// If there is an error unmarshaling the JSON, the configuration is left unchanged.
func loadConfig() {
	b, err := os.ReadFile(ConfigPath())
	if err == nil {
		_ = json.Unmarshal(b, &config)
	}

	if config.RigBaud == 0 {
		config.RigBaud = 9600
	}

	// 後方互換性: 既存のRigPort/RigBaudをRigPorts[0]にマイグレーション
	if len(config.RigPorts) == 0 && config.RigPort != "" {
		config.RigPorts = []RigPortConfig{
			{Port: config.RigPort, Baud: config.RigBaud},
		}
	}

	// RigPortsが空の場合は4つの空エントリで初期化
	if len(config.RigPorts) == 0 {
		config.RigPorts = make([]RigPortConfig, 4)
		for i := range config.RigPorts {
			config.RigPorts[i].Baud = 9600
		}
	}

	// 4つに満たない場合は拡張
	for len(config.RigPorts) < 4 {
		config.RigPorts = append(config.RigPorts, RigPortConfig{Baud: 9600})
	}

	// デフォルト値
	if config.RigBroadcastMode == "" {
		config.RigBroadcastMode = "all"
	}
}

// saveConfig saves the current configuration to a file named
// "config.json". It marshals the configuration into JSON and
// writes it to the file with permissions 0600.
func saveConfig() {
	b, _ := json.MarshalIndent(config, "", "  ")
	_ = os.WriteFile(ConfigPath(), b, 0600)
}

func ConfigPath() string {
	return filepath.Join(appDataDir(), "config.json")
}
