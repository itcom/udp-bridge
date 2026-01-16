package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	QRZUser string `json:"qrz_user"`
	QRZPass string `json:"qrz_pass"`

	UseQRZ bool `json:"use_qrz"`
	UseGeo bool `json:"use_geo"`

	UseRig  bool   `json:"use_rig"`
	RigPort string `json:"rig_port"`
	RigBaud int    `json:"rig_baud"`
	UsePTY  bool   `json:"use_pty"`
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
