package main

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	err := rdb.Del(ctx, "events:recent").Err()
	if err != nil {
		fmt.Printf("Error deleting key: %v\n", err)
	} else {
		fmt.Println("Successfully deleted events:recent")
	}
}
