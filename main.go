package main

import (
	"analytics-backend/database"
	"analytics-backend/handlers"
	"analytics-backend/worker"
	"context"
	"github.com/gin-gonic/gin"
	"log"
)

var ctx = context.Background()

func main() {
	database.InitRedis()
	database.Initdb()
	if err := database.EnsureConsumerGroup(); err != nil {
		log.Fatalf("Failed to create consumer group: %v", err)
	}

	log.Println("✅ Consumer group created successfully")

	go worker.StartWorker()

	router := gin.Default()
	router.POST("/event", handlers.GetEvent)
	router.GET("/events", handlers.FetchEvents)
	router.Run(":8080")

}
