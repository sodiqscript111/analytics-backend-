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

func EnsureConsumerGroup() error {
	err := Rdb.XGroupCreateMkStream(Ctx, StreamName, GroupName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func ReadFromGroup() ([]redis.XMessage, error) {
	results, err := Rdb.XReadGroup(Ctx, &redis.XReadGroupArgs{
		Group:    GroupName,
		Consumer: Consumer,
		Streams:  []string{StreamName, ">"},
		Count:    BatchSize,
		Block:    BlockTimeMs,
	}).Result()

	if err != nil {

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

func AckMessage(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return Rdb.XAck(Ctx, StreamName, GroupName, ids...).Err()
}

func CheckStreamLength(stream string) (int64, error) {
	return Rdb.XLen(Ctx, stream).Result()
}
