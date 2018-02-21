package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/andrew-boutin/go-chat/db"
	"github.com/andrew-boutin/go-chat/user"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Inside the compose network we can use the service name for the address
var redisAddr = "redis:6379"

// WebSocket members
// Map of WebSocket pointers
var clients = make(map[*websocket.Conn]string)

// Channel queue for incoming messages from clients
var broadcast = make(chan Message)

// Used to upgrade HTTP connections to WebSockets
var upgrader = websocket.Upgrader{}

// Auth members
var cred Credentials
var conf *oauth2.Config
var state string

// Message holds a chat message
type Message struct {
	// Backticks inform Go on how to marshall struct/JSON
	Username string `json:"username"`
	Message  string `json:"message"`
}

// Credentials holds the client ID and secret
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

	db.InitClient(redisAddr)
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
func loginHandler(c *gin.Context) {
	// Create state info, store in a session, and generate the Google URL for the user to begin login
	state = randToken()
	session := sessions.Default(c)
	session.Set("state", state)
	session.Save()
	c.Writer.Write([]byte("<html><body>Go-Chat<br><a href='" + getLoginURL(state) + "'>Login with Google</a><br>to start chatting!</body></html>"))
}

// Handle user redirect back from Google login to get oauth token
func authHandler(c *gin.Context) {
	// Get the state from the User's request
	session := sessions.Default(c)
	retrievedState := session.Get("state")

	// TODO: How is the session cleanup handled?
	if retrievedState != c.Query("state") {
		log.Printf("No matching state found for the session, not accepting auth request.")
		c.Redirect(302, "/login")
		// TODO: c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session state: %s", retrievedState))?
		return
	}

	tok, err := conf.Exchange(oauth2.NoContext, c.Query("code"))

	if err != nil {
		// TODO: c.AbortWithError(http.StatusBadRequest, err)?
		log.Fatal(err)
		return
	}

	client := conf.Client(oauth2.NoContext, tok)

	googleUserData, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		log.Fatal(err)
		return // TODO:
	}
	defer googleUserData.Body.Close()
	data, _ := ioutil.ReadAll(googleUserData.Body)

	var user user.User
	err = json.Unmarshal(data, &user)

	if err != nil {
		log.Fatal(err)
		return // TODO:
	}

	db.StoreUser(user)

	session.Set("user-id", user.ID)
	err = session.Save()

	if err != nil {
		log.Fatal(err)
		return // TODO:
	}

	c.Redirect(302, "/")
}

// Take an HTTP request and upgrade it to a WebSocket
func handleConnection(c *gin.Context) {
	// Upgrade from HTTP request to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		log.Fatal(err)
	}

	// Close the WebSocket when exiting the function
	defer ws.Close()

	session := sessions.Default(c)
	v := session.Get("user-id")
	user := db.GetStoredUser(v.(string))
	email := user.Email
	welcomeMsg := Message{"server", "Hello " + email}
	err = ws.WriteJSON(welcomeMsg)

	if err != nil {
		// TODO:
		log.Fatal(err)
	}

	// Map the connection for tracking
	clients[ws] = email

	emails := make([]string, 0, len(clients))
	for _, v := range clients {
		emails = append(emails, v)
	}
	users := strings.Join(emails, ",")
	usersMsg := Message{"users", users}
	//err = ws.WriteJSON(usersMsg)
	broadcast <- usersMsg

	if err != nil {
		// TODO:
		log.Fatal(err)
	}

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
		for ws, email := range clients {
			// Could be a system message that already defines the username
			if msg.Username == "" {
				msg.Username = email
			}

			err := ws.WriteJSON(msg)

			// Get rid of the connection on errors
			if err != nil {
				log.Printf("error: %v", err)
				ws.Close()
				delete(clients, ws)
			}
		}
	}
}

// Gin Middleware that requires the user to be authenticated in order to go to certain routes
func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		v := session.Get("user-id")

		// If this is a new session then redirect to the login page
		if v == nil {
			c.Redirect(302, "/login")
			return
		}

		c.Next()
	}
}

func main() {
	// Get our Gin engine
	r := gin.Default()

	// Create the Redis session store
	store, err := sessions.NewRedisStore(10, "tcp", redisAddr, "", []byte("secret"))

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Set up Gin to use Redis session store
	r.Use(sessions.Sessions("mysession", store))

	// Protect our endpoints w/ custom middleware
	authorized := r.Group("/")
	authorized.Use(authRequired())
	{
		// Allow js and css static files to be accessed
		authorized.Static("/static", "./static")

		// Set up serving the site index
		authorized.StaticFile("/", "./html")

		// Set up entry point for WebSocket connections
		authorized.GET("/ws", handleConnection)
	}

	// Auth handlers
	r.GET("/login", loginHandler)
	r.GET("/auth", authHandler)

	// Start concurrent process for handling incoming connections
	go handleMessages()

	// Start up the server
	log.Printf("Starting server.")
	r.Run(":8080")
	log.Printf("Server exited")
}
