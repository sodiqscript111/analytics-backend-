package database

import (
	"analytics-backend/models"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	StreamName  = "events"
	GroupName   = "event-group"
	Consumer    = "worker-1"
	BatchSize   = int64(100)
	BlockTimeMs = 300 * time.Millisecond
)

// AddToStream adds a new event into the Redis stream.
func AddToStream(stream models.Event) error {
	_, err := Rdb.XAdd(Ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{
			"user_id":   stream.UserId,
			"action":    stream.Action,
			"element":   stream.Element,
			"duration":  stream.Duration,
			"timestamp": stream.Timestamp,
		},
	}).Result()
	return err
}

// EnsureConsumerGroup creates the consumer group if it doesn’t already exist.
func EnsureConsumerGroup() error {
	// XGroupCreateMkStream ensures the stream exists, creates group if not exists
	err := Rdb.XGroupCreateMkStream(Ctx, StreamName, GroupName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// ReadFromGroup reads messages from the Redis stream as part of a consumer group.
func ReadFromGroup() ([]redis.XMessage, error) {
	results, err := Rdb.XReadGroup(Ctx, &redis.XReadGroupArgs{
		Group:    GroupName,
		Consumer: Consumer,
		Streams:  []string{StreamName, ">"},
		Count:    BatchSize,
		Block:    BlockTimeMs,
	}).Result()

	if err != nil {
		// If no messages within block timeout, return empty
		if err == redis.Nil {
			return []redis.XMessage{}, nil
		}
		return nil, err
	}

	var messages []redis.XMessage
	for _, stream := range results {
		messages = append(messages, stream.Messages...)
	}
	return messages, nil
}

// AckMessage acknowledges one or more message IDs so Redis can delete them from the PEL.
func AckMessage(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return Rdb.XAck(Ctx, StreamName, GroupName, ids...).Err()
}

// CheckStreamLength returns the current length of the stream.
func CheckStreamLength(stream string) (int64, error) {
	return Rdb.XLen(Ctx, stream).Result()
}
