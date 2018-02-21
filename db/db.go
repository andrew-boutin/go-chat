package db

import (
	"encoding/json"
	"log"
	"os"

	"github.com/andrew-boutin/go-chat/user"

	"github.com/go-redis/redis"
)

var redisClient *redis.Client

// InitClient sets up the DB connection to the Redis store
func InitClient(redisAddr string) {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0, // Default
	})

	_, err := redisClient.Ping().Result()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

// GetStoredUser retrieves a User from the Redis store
func GetStoredUser(userID string) user.User {
	var user user.User
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

// StoreUser stores the User data into the Redis store
func StoreUser(user user.User) {
	userAsJSON, err := json.Marshal(user)

	if err != nil {
		log.Fatal("Failed to store user in store.")
		return
	}

	redisClient.Set("user:"+user.ID, string(userAsJSON), 0)
}
