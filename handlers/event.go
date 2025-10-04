package handlers

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"analytics-backend/worker"
	"github.com/gin-gonic/gin"
)

func GetEvent(c *gin.Context) {
	var event models.Event
	if err := c.ShouldBind(&event); err != nil {
		c.JSON(400, gin.H{"There was an error": err.Error()})
		return
	}
	if err := database.AddToStream(event); err != nil {
		c.JSON(500, gin.H{"There was an error": err.Error()})
		return
	}
	worker.AddToDatabase()
	c.JSON(200, gin.H{"event": event})
}
