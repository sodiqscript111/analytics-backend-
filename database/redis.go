package database

import (
	"analytics-backend/config"
	"context"
	"log"
	"time"

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

func PushToRecentFeed(ctx context.Context, eventJSON []byte, snowflakeID int64) error {
	started := time.Now()
	pipe := Rdb.Pipeline()

	pipe.ZAdd(ctx, "events:recent", redis.Z{
		Score:  float64(snowflakeID),
		Member: eventJSON,
	})

	pipe.ZRemRangeByRank(ctx, "events:recent", 0, -51)

	_, err := pipe.Exec(ctx)
	observeRedisOperation("push_recent_feed", "events:recent", started, err)

	if err != nil && err.Error() == "WRONGTYPE Operation against a key holding the wrong kind of value" {
		log.Println("Detected key type mismatch for events:recent, deleting old key...")
		Rdb.Del(ctx, "events:recent")
		return PushToRecentFeed(ctx, eventJSON, snowflakeID)
	}

	return err
}

func GetRecentFeed(ctx context.Context) ([]string, error) {
	started := time.Now()
	results, err := Rdb.ZRevRange(ctx, "events:recent", 0, 49).Result()
	observeRedisOperation("read_recent_feed", "events:recent", started, err)
	return results, err
}

func PublishEvent(ctx context.Context, eventJSON []byte) error {
	started := time.Now()
	err := Rdb.Publish(ctx, "events:stream", eventJSON).Err()
	observeRedisOperation("publish_event", "events:stream", started, err)
	return err
}
