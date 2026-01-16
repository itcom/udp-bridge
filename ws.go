// ws.go
package main

import (
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
