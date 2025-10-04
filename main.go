package main

import (
	"analytics-backend/database"
	"analytics-backend/handlers"
	"context"
	"github.com/gin-gonic/gin"
)

var ctx = context.Background()

func main() {
	database.InitRedis()
	database.Initdb()
	router := gin.Default()
	router.POST("/event", handlers.GetEvent)
	router.Run(":8080")

}
