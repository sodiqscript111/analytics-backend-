package database

import (
	"analytics-backend/models"
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	IndexStreamName  = "events:index"
	IndexGroupName   = "event-indexers"
	IndexBatchSize   = int64(200)
	IndexBlockTimeMs = 300 * time.Millisecond
)

func EnsureIndexerGroup() error {
	err := Rdb.XGroupCreateMkStream(Ctx, IndexStreamName, IndexGroupName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func EnqueueEventsForIndexing(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}

	started := time.Now()
	pipe := Rdb.Pipeline()
	for _, event := range events {
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: IndexStreamName,
			Values: map[string]any{
				"id":        event.ID,
				"user_id":   event.UserId,
				"action":    event.Action,
				"element":   event.Element,
				"duration":  event.Duration,
				"timestamp": event.Timestamp.UTC().Format(time.RFC3339Nano),
			},
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Failed to enqueue index jobs: %v", err)
	}
	observeRedisOperation("enqueue_index_jobs", IndexStreamName, started, err)
	return err
}

func ReadIndexJobsFromGroup(consumer string) ([]redis.XMessage, error) {
	started := time.Now()
	results, err := Rdb.XReadGroup(Ctx, &redis.XReadGroupArgs{
		Group:    IndexGroupName,
		Consumer: consumer,
		Streams:  []string{IndexStreamName, ">"},
		Count:    IndexBatchSize,
		Block:    IndexBlockTimeMs,
		NoAck:    false,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			observeRedisOperation("read_index_jobs", IndexStreamName, started, nil)
			return []redis.XMessage{}, nil
		}
		log.Printf("Failed to read index jobs: %v", err)
		observeRedisOperation("read_index_jobs", IndexStreamName, started, err)
		return nil, err
	}

	var messages []redis.XMessage
	for _, stream := range results {
		messages = append(messages, stream.Messages...)
	}
	observeRedisOperation("read_index_jobs", IndexStreamName, started, nil)
	return messages, nil
}

func AckIndexJobs(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	started := time.Now()
	err := Rdb.XAck(Ctx, IndexStreamName, IndexGroupName, ids...).Err()
	if err != nil {
		log.Printf("Failed to acknowledge index jobs: %v", err)
	}
	observeRedisOperation("ack_index_jobs", IndexStreamName, started, err)
	return err
}
