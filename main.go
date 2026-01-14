package main

import "log"

// main starts the HAMLAB Bridge. It loads the configuration from a file named
// "config.json", starts the settings UI web server, and starts the bridge
// server which listens for incoming WSJT-X/JTDX messages and broadcasts
// them to connected WebSocket clients.
func main() {
	loadConfig()

	go startWebUI()
	log.Println("Settings UI: http://127.0.0.1:17801/settings")

	startBridge()
}
