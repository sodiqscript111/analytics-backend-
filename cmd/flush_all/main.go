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
	// FlushDB flushes the currently selected database (0)
	err := rdb.FlushDB(ctx).Err()
	if err != nil {
		fmt.Printf("Error flushing DB: %v\n", err)
	} else {
		fmt.Println("Successfully flushed Redis DB")
	}
}
