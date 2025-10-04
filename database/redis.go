package database

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	Ctx = context.Background()
	Rdb *redis.Client
)

// InitRedis initializes and tests the Redis connection
func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // change if using Docker container
		Password: "",               // set if you use a password
		DB:       0,                // default DB
	})

	// Test connection
	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	log.Println("✅ Connected to Redis")
}
