package main

import (
	"os"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

// Map of WebSocket pointers
var clients = make(map[*websocket.Conn]bool)

// Channel queue for incoming messages from clients
var broadcast = make(chan Message)

// Used to upgrade HTTP connections to WebSockets
var upgrader = websocket.Upgrader{}

type Message struct {
	// Backticks inform Go on how to marshall struct/JSON
	Username string `json:"username"`
	Message string `json:"message"`
}

// Basic initialization
func init() {
	log.SetOutput(os.Stdout)
}

// Take an HTTP request and upgrade it to a WebSocket
func handleConnection(w http.ResponseWriter, req *http.Request) {
	// Upgrade from HTTP request to WebSocket
	ws, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		log.Fatal(err)
	}

	// Close the WebSocket when exiting the function
	defer ws.Close()

	// Map the connection for tracking
	clients[ws] = true

	for {
		var msg Message

		// Pass in a pointer to the Message to preserve data
		err := ws.ReadJSON(&msg)

		// Get rid of the connection on errors and exit
		if err != nil {
			// Log the error using the default format
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}

		// Send the message to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// Get the next message out of the broadcast channel
		msg := <-broadcast

		// Send the message to all of the registered clients
		for ws := range clients {
			err := ws.WriteJSON(msg)

			// Get rid of the connection on errors
			if err != nil {
				log.Printf("handleMessages erroring out")
				log.Printf("error: %v", err)
				ws.Close();
				delete(clients, ws)
			}
		}
	}
}

// Program entry point
func main() {
	// Allow js and css static files to be accessed
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Set up serving the site index
	http.Handle("/", http.FileServer(http.Dir("./html")))

	// Set up entry point for WebSocket connections
	http.HandleFunc("/ws", handleConnection)

	// Start concurrent process for handling incoming connections
	go handleMessages()

	// Start up the server
	log.Printf("Starting server.")
	log.Fatal(http.ListenAndServe(":8080", nil))
	log.Printf("Server exited")
}
