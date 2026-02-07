package utils

import (
	"log"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// InitSnowflake initializes the snowflake node.
// nodeID should be unique for each running instance of the application (0-1023).
func InitSnowflake(nodeID int64) {
	once.Do(func() {
		var err error
		node, err = snowflake.NewNode(nodeID)
		if err != nil {
			log.Fatalf("Failed to initialize snowflake node: %v", err)
		}
	})
}

// GenerateID returns a new snowflake ID.
func GenerateID() int64 {
	if node == nil {
		log.Println("Snowflake node not initialized, initializing with default node ID 1")
		InitSnowflake(1)
	}
	return node.Generate().Int64()
}
