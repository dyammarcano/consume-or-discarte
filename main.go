package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/oklog/ulid/v2"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const logChannelSize = 100

type Client struct {
	Conn *websocket.Conn
}

var (
	logMessages = make(chan string, logChannelSize)
	clients     map[*Client]bool
	mutex       = &sync.Mutex{}

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clients = make(map[*Client]bool)
	logMessages = make(chan string)

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", handleIndex)

	log.Println("Starting log streaming server on port 8080...")
	server := &http.Server{Addr: ":8080"}

	go generateLogMessages(logMessages)
	go broadcastLogMessagesToClients(ctx, logMessages)

	startServerContext(server, ctx)
}

func broadcastLogMessagesToClients(ctx context.Context, logMessages chan string) {
	defer clear(clients)

	for {
		select {
		case msg := <-logMessages:
			fmt.Fprintln(os.Stdout, msg)
			mutex.Lock()
			for client := range clients {
				err := client.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					log.Printf("Websocket error: %s", err)
					client.Conn.Close()
					delete(clients, client)
				}
			}
			mutex.Unlock()
		case <-ctx.Done():
			log.Println("Stopping log message broadcast")
			return
		}
	}
}

func startServerContext(server *http.Server, ctx context.Context) {
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	<-ctx.Done()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	close(logMessages)

	log.Print("Server Exited Properly")
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error upgrading WebSocket", http.StatusInternalServerError)
		return
	}
	//defer conn.Close()

	client := &Client{Conn: conn}

	// starting from here
	mutex.Lock()
	clients[client] = true
	mutex.Unlock()
	// to here

	//defer func() {
	//	mutex.Lock() // We also lock on delete to prevent concurrent access
	//	delete(clients, client)
	//	mutex.Unlock()
	//}()
}

func generateLogMessages(logMessages chan string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for t := range ticker.C {
		msg := fmt.Sprintf("hash: %s, number: %v", ulid.Make().String(), t.Format(time.RFC3339))
		logMessages <- msg
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WebSocket Log Consumer</title>
    <style>
        body {
            font-family: Arial, sans-serif;
        }
        #log-container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            border: 1px solid #ccc;
            border-radius: 5px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            background-color: #f9f9f9;
        }
        .log-message {
            margin-bottom: 10px;
            padding: 10px;
            background-color: #ffffff;
            border: 1px solid #ccc;
            border-radius: 5px;
        }
        .error-message {
            color: red;
        }
    </style>
</head>

<body>
    <div id="log-container"></div>

    <script>
        const logContainer = document.getElementById("log-container");
        const socket = new WebSocket("ws://{{.Host}}:{{.Port}}/ws");

        socket.onopen = function (event) {
            console.log("WebSocket connection opened:", event);
        };

        socket.onmessage = function (event) {
            const logMessage = event.data;
            if (!logMessage) { 
                console.error("Empty logMessage received");
                return;
            }
            const logElement = document.createElement("div");
            logElement.textContent = logMessage;
            logElement.className = "log-message";
            logContainer.appendChild(logElement);
        };

        socket.onclose = function (event) {
            if (event.wasClean) {
                console.log("Closed cleanly, code=${event.code}, reason=${event.reason}");
            } else {
                console.error("Connection died");
            }
        };

        socket.onerror = function (error) {
            console.error("WebSocket Error:", error);
        };
    </script>
</body>

</html>
`

	tmpl := template.Must(template.New("index").Parse(htmlTemplate))
	data := struct {
		Host string
		Port string
	}{
		Host: "localhost",
		Port: "8080",
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		return
	}
}
