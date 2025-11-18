package handlers

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"context"
	"github.com/gin-gonic/gin"
	"time"
)

func GetEvent(c *gin.Context) {
	var event models.Event
	if err := c.ShouldBind(&event); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := database.AddToStreamWithContext(ctx, event); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"status": "accepted", "event": event})
}
