package database

import (
	"analytics-backend/config"
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	Ctx = context.Background()
	Rdb *redis.Client
)

func InitRedis(cfg config.RedisConfig) {
	Rdb = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   3,
	})

	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", cfg.Addr, err)
	}

	log.Println("Connected to Redis")
}

// Recent Feed Helpers

func PushToRecentFeed(ctx context.Context, eventJSON []byte, snowflakeID int64) error {
	pipe := Rdb.Pipeline()

	// Add to Sorted Set with Snowflake ID as score
	// Snowflake IDs are integers that are time-ordered.
	pipe.ZAdd(ctx, "events:recent", redis.Z{
		Score:  float64(snowflakeID),
		Member: eventJSON,
	})

	// Keep only top 10 most recent events
	// ZRemRangeByRank removes elements by rank (0-based)
	// We want to keep the last 10 (highest scores), so we remove from 0 to -11
	pipe.ZRemRangeByRank(ctx, "events:recent", 0, -11)

	_, err := pipe.Exec(ctx)

	// Handle potential key type mismatch if switching from List to ZSet
	if err != nil && err.Error() == "WRONGTYPE Operation against a key holding the wrong kind of value" {
		log.Println("Detected key type mismatch for events:recent, deleting old key...")
		Rdb.Del(ctx, "events:recent")
		return PushToRecentFeed(ctx, eventJSON, snowflakeID)
	}

	return err
}

func GetRecentFeed(ctx context.Context) ([]string, error) {
	// Get top 10 most recent events (highest scores -> newest events)
	return Rdb.ZRevRange(ctx, "events:recent", 0, 9).Result()
}
