// ws.go
package main

import (
	"net/http"

	"github.com/gorilla/websocket"

	"log"
)

var clients = map[*websocket.Conn]bool{}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsHandler upgrades the HTTP connection to a WebSocket connection.
// It stores the WebSocket connection in the clients map for later use.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := upgrader.Upgrade(w, r, nil)
	clients[c] = true
}

// Broadcast sends a message to all connected WebSocket clients.
// If any error occurs when sending to a client, that client is removed from the clients map and closed.
func broadcast(msg string) {
	log.Println("[WS] broadcast:", msg)
	for c := range clients {
		if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			delete(clients, c)
			c.Close()
		}
	}
}

// startWebSocket starts a WebSocket server on localhost:17800.
// It upgrades incoming HTTP connections to WebSocket connections and stores them in the clients map.
// When a message is sent to the broadcast function, it is sent to all connected WebSocket clients.
func startWebSocket() {
	http.HandleFunc("/ws", wsHandler)
	http.ListenAndServe("127.0.0.1:17800", nil)
}
