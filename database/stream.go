package database

import (
	"analytics-backend/models"
	"github.com/redis/go-redis/v9"
)

func AddToStream(stream models.Event) error {
	_, err := Rdb.XAdd(Ctx, &redis.XAddArgs{
		Stream: "events",
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

func CheckStreamlenght(stream string) (int64, error) {
	return Rdb.XLen(Ctx, stream).Result()

}

func ReadLenght() {
	_, err := Rdb.XRead(Ctx, &redis.XReadArgs{
		Streams: []string{"race:france", "0"},
		Count:   100,
		Block:   300,
	}).Result()

	if err != nil {
		panic(err)
	}
}
