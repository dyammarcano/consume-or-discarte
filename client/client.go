package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/url"
)

func main() {
	// Connecting to the WebSocket server
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to the WebSocket Server: %v", err)
	}
	defer c.Close()

	// Handle incoming WebSocket messages
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Connection died")
			}
			break
		}

		if len(message) == 0 {
			log.Printf("Empty logMessage received")
			continue
		}

		log.Printf("WebSocket message received: %s", message)
	}
}
