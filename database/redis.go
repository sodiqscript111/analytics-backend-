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

func InitRedis() {

	addr := "redis:6379"
	Rdb = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     500,
		MinIdleConns: 50,
		MaxRetries:   3,
	})

	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to redis:6379, trying localhost:6379")
		Rdb = redis.NewClient(&redis.Options{
			Addr:         "localhost:6379",
			Password:     "",
			DB:           0,
			PoolSize:     500,
			MinIdleConns: 50,
			MaxRetries:   3,
		})
		_, err = Rdb.Ping(Ctx).Result()
		if err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
	}

	log.Println("Connected to Redis")
}

// Recent Feed Helpers

func PushToRecentFeed(ctx context.Context, eventJSON []byte) error {
	pipe := Rdb.Pipeline()

	// Push to head of list
	pipe.LPush(ctx, "events:recent", eventJSON)
	// Keep only top 10
	pipe.LTrim(ctx, "events:recent", 0, 9)

	_, err := pipe.Exec(ctx)
	return err
}

func GetRecentFeed(ctx context.Context) ([]string, error) {
	return Rdb.LRange(ctx, "events:recent", 0, 9).Result()
}
