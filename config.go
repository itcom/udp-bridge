package main

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	QRZUser string `json:"qrz_user"`
	QRZPass string `json:"qrz_pass"`

	UseQRZ bool `json:"use_qrz"`
	UseGeo bool `json:"use_geo"`
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
	b, err := os.ReadFile("config.json")
	if err == nil {
		_ = json.Unmarshal(b, &config)
	}
}

// saveConfig saves the current configuration to a file named
// "config.json". It marshals the configuration into JSON and
// writes it to the file with permissions 0600.
func saveConfig() {
	b, _ := json.MarshalIndent(config, "", "  ")
	_ = os.WriteFile("config.json", b, 0600)
}
