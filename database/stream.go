package database

import (
	"analytics-backend/models"
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	StreamName  = "events"
	GroupName   = "event-group"
	BatchSize   = int64(1000)
	BlockTimeMs = 300 * time.Millisecond
)

func AddToStream(stream models.Event) error {
	return AddToStreamWithContext(Ctx, stream)
}

func AddToStreamWithContext(ctx context.Context, stream models.Event) error {
	started := time.Now()
	_, err := Rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{
			"id":        stream.ID,
			"user_id":   stream.UserId,
			"action":    stream.Action,
			"element":   stream.Element,
			"duration":  stream.Duration,
			"timestamp": stream.Timestamp.Format(time.RFC3339Nano),
		},
	}).Result()

	if err != nil {
		log.Printf("Failed to add to stream: %v", err)
	}
	observeRedisOperation("add_to_stream", StreamName, started, err)

	return err
}

func EnsureConsumerGroup() error {
	err := Rdb.XGroupCreateMkStream(Ctx, StreamName, GroupName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func ReadFromGroup(consumer string) ([]redis.XMessage, error) {
	started := time.Now()
	results, err := Rdb.XReadGroup(Ctx, &redis.XReadGroupArgs{
		Group:    GroupName,
		Consumer: consumer,
		Streams:  []string{StreamName, ">"},
		Count:    BatchSize,
		Block:    BlockTimeMs,
		NoAck:    false,
	}).Result()

	if err != nil {

		if err == redis.Nil {
			observeRedisOperation("read_group", StreamName, started, nil)
			return []redis.XMessage{}, nil
		}
		log.Printf("Failed to read from group: %v", err)
		observeRedisOperation("read_group", StreamName, started, err)
		return nil, err
	}

	var messages []redis.XMessage
	for _, stream := range results {
		messages = append(messages, stream.Messages...)
	}
	observeRedisOperation("read_group", StreamName, started, nil)
	return messages, nil
}

func AckMessage(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	started := time.Now()
	err := Rdb.XAck(Ctx, StreamName, GroupName, ids...).Err()
	if err != nil {
		log.Printf("Failed to acknowledge messages: %v", err)
	}
	observeRedisOperation("ack_group", StreamName, started, err)

	return err
}

func CheckStreamLength(stream string) (int64, error) {
	started := time.Now()
	length, err := Rdb.XLen(Ctx, stream).Result()
	if err != nil {
		log.Printf("Failed to check stream length: %v", err)
	}
	observeRedisOperation("stream_length", stream, started, err)
	return length, err
}
