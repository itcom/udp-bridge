package main

import (
	"log"
	"os"
	"path/filepath"
)

// main starts the HAMLAB Bridge. It loads the configuration from a file named
// "config.json", starts the settings UI web server, and starts the bridge
// server which listens for incoming WSJT-X/JTDX messages and broadcasts
// them to connected WebSocket clients.
func main() {
	log.Println("App data dir:", appDataDir())
	loadConfig()

	go startWebUI()

	setupLaunchAgent()
	go startWebSocket()
	go startBridge()
	go startRigWatcher()

	select {}
}

// appDataDir returns the path to the HAMLAB Bridge's app data directory.
// The directory is created if it does not exist.
// The function logs a fatal error if it cannot get the user's config directory or create the app data directory.
// The returned path is the full path to the app data directory, including the directory name "HAMLAB Bridge".
func appDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	dir := filepath.Join(base, "HAMLAB Bridge")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}

	return dir
}
