package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// WebSocket members
// Map of WebSocket pointers
var clients = make(map[*websocket.Conn]bool)

// Channel queue for incoming messages from clients
var broadcast = make(chan Message)

// Used to upgrade HTTP connections to WebSockets
var upgrader = websocket.Upgrader{}

// Auth members
var cred Credentials
var conf *oauth2.Config
var state string

type Message struct {
	// Backticks inform Go on how to marshall struct/JSON
	Username string `json:"username"`
	Message  string `json:"message"`
}

// Client Id / Secret Credentials
type Credentials struct {
	Cid     string `json:"cid"`
	Csecret string `json:"csecret"`
}

// Basic initialization
func init() {
	log.SetOutput(os.Stdout)

	// Read the credentials out from the file
	file, err := ioutil.ReadFile("./creds.json")

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Load up the credentials
	json.Unmarshal(file, &cred)

	conf = &oauth2.Config{
		ClientID:     cred.Cid,
		ClientSecret: cred.Csecret,
		RedirectURL:  "http://localhost:8080/auth",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func getLoginURL(state string) string {
	return conf.AuthCodeURL(state)
}

// Handle user attempts to login
func loginHandler(w http.ResponseWriter, req *http.Request) {
	// Create state info, store in a session, and generate the Google URL for the user to begin login
	state = randToken()
	// TODO: Save the state in a session
	w.Write([]byte("<html><body>Go Chat<a href='" + getLoginURL(state) + "'>Login with Google</a></body></html>"))
}

// Handle user redirect back from Google login to get oauth token
func authHandler(w http.ResponseWriter, req *http.Request) {
	// TODO: compare to stored state info
	// s = req.URL.Query().Get("state")

	tok, err := conf.Exchange(oauth2.NoContext, req.URL.Query().Get("code"))

	if err != nil {
		// TODO: Handle bad request here
		log.Fatal(err)
		return
	}

	client := conf.Client(oauth2.NoContext, tok)

	email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		log.Fatal()
		return
	}
	defer email.Body.Close()
	data, _ := ioutil.ReadAll(email.Body)
	log.Println("Email body: ", string(data))
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
				ws.Close()
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

	// Auth handlers
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/auth", authHandler)

	// Start concurrent process for handling incoming connections
	go handleMessages()

	// Start up the server
	log.Printf("Starting server.")
	log.Fatal(http.ListenAndServe(":8080", nil))
	log.Printf("Server exited")
}
