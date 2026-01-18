// ws.go
package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"log"
)

var clients = map[*websocket.Conn]bool{}
var clientsMu sync.Mutex

var broadcastChan = make(chan string, 100)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsHandler upgrades the HTTP connection to a WebSocket connection.
// It stores the WebSocket connection in the clients map for later use.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := upgrader.Upgrade(w, r, nil)
	clientsMu.Lock()
	clients[c] = true
	clientsMu.Unlock()

	// Start goroutine to handle incoming messages from this client
	go handleClientMessages(c)
}

// Broadcast sends a message to all connected WebSocket clients.
// If any error occurs when sending to a client, that client is removed from the clients map and closed.
func broadcast(msg string) {
	select {
	case broadcastChan <- msg:
	default:
		// バッファフル時は破棄（シリアル読み取りをブロックしない）
	}
}

// broadcastWorker processes messages from the broadcast channel.
func broadcastWorker() {
	for msg := range broadcastChan {
		// log.Println("[WS] broadcast:", msg)
		clientsMu.Lock()
		for c := range clients {
			if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				delete(clients, c)
				c.Close()
			}
		}
		clientsMu.Unlock()
	}
}

// handleClientMessages handles incoming messages from a WebSocket client
func handleClientMessages(c *websocket.Conn) {
	defer func() {
		clientsMu.Lock()
		delete(clients, c)
		c.Close()
		clientsMu.Unlock()
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}

		// Parse incoming message as JSON
		var req map[string]interface{}
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}

		// Handle different message types
		if msgType, ok := req["type"].(string); ok {
			switch msgType {
			case "getRigState":
				// Get RigState for a specific port
				portIndex := -1
				if idx, ok := req["port"].(float64); ok {
					portIndex = int(idx)
				}

				var response map[string]interface{}
				if portIndex >= 0 {
					// Get specific port state
					rigStatesMu.RLock()
					if state, exists := rigStates[portIndex]; exists && state != nil {
						response = map[string]interface{}{
							"type":  "rigState",
							"port":  portIndex,
							"freq":  state.Freq,
							"mode":  state.Mode,
							"data":  state.Data,
							"proto": state.Proto,
						}
					} else {
						response = map[string]interface{}{
							"type":  "error",
							"error": "Port not found or not initialized",
						}
					}
					rigStatesMu.RUnlock()
				} else {
					// Get all port states
					rigStatesMu.RLock()
					states := make(map[string]interface{})
					for idx, state := range rigStates {
						if state != nil {
							states[string(rune(idx+'0'))] = map[string]interface{}{
								"freq":  state.Freq,
								"mode":  state.Mode,
								"data":  state.Data,
								"proto": state.Proto,
								"port":  state.Index,
							}
						}
					}
					response = map[string]interface{}{
						"type":   "rigStates",
						"states": states,
					}
					rigStatesMu.RUnlock()
				}

				// Send response
				if responseBytes, err := json.Marshal(response); err == nil {
					c.WriteMessage(websocket.TextMessage, responseBytes)
				}
			}
		}
	}
}

// startWebSocket starts a WebSocket server on localhost:17800.
// It upgrades incoming HTTP connections to WebSocket connections and stores them in the clients map.
// When a message is sent to the broadcast function, it is sent to all connected WebSocket clients.
func startWebSocket() {
	go broadcastWorker()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket: 127.0.0.1:17800/ws")
	http.ListenAndServe("127.0.0.1:17800", mux)
}
