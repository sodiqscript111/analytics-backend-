package handlers

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"github.com/gin-gonic/gin"
)

func GetEvent(c *gin.Context) {
	var event models.Event
	if err := c.ShouldBind(&event); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := database.AddToStream(event); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"status": "accepted", "event": event})
}
