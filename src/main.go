package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Redis
var redisClient *redis.Client

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

type User struct {
	ID                       string `json:"sub"`
	Name                     string `json:"name"`
	FirstName                string `json:"given_name`
	LastName                 string `json:"family_name`
	GoogleProfilePictureLink string `json:"picture"`
	Email                    string `json:"email"`
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

	// Inside the compose network we can use the service name for the address
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0, // Default
	})

	pong, err := redisClient.Ping().Result()
	log.Printf(pong, err)
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
	redisClient.Set(state, false, time.Minute)
	w.Write([]byte("<html><body>Go-Chat<br><a href='" + getLoginURL(state) + "'>Login with Google</a><br>to start chatting!</body></html>"))
}

// Handle user redirect back from Google login to get oauth token
func authHandler(w http.ResponseWriter, req *http.Request) {
	// Get the state from the User's request
	var s = req.URL.Query().Get("state")

	_, err := redisClient.Get(s).Result()

	if err == redis.Nil {
		log.Printf("No matching state in the store, not accepting auth request.")
		http.Redirect(w, req, "/login", 302)
		return
	} else if err != nil {
		log.Fatal(err)
		http.Redirect(w, req, "/login", 302)
		return
	} else {
		log.Printf("Found state in store, completing auth request.")
		redisClient.Set(state, true, 0)
		// TODO: State info cleanup...
	}

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

	// TODO: Require auth to go to pages..

	var user User
	err = json.Unmarshal(data, &user)
	log.Printf("User from google api: %+v", user)

	storeUser(user)
}

// Get a User from Redis
func getStoredUser(userID string) User {
	var user User
	userAsString, err := redisClient.Get("user:" + userID).Result()

	if err != nil {
		log.Fatal("Failed to read user from store.", err)
		return user
	}

	err = json.Unmarshal([]byte(userAsString), &user)

	if err != nil {
		log.Fatal("Failed to convert string to User.", err)
		return user
	}

	return user
}

// Store the User info in Redis
func storeUser(user User) {
	userAsJson, err := json.Marshal(user)

	if err != nil {
		log.Fatal("Failed to store user in store.")
		return
	}

	redisClient.Set("user:"+user.ID, string(userAsJson), 0)
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
