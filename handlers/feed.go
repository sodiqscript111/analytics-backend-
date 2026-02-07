package handlers

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

func GetRecentFeed(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rawEvents, err := database.GetRecentFeed(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var events []models.Event
	for _, raw := range rawEvents {
		var e models.Event
		if err := json.Unmarshal([]byte(raw), &e); err == nil {
			events = append(events, e)
		}
	}

	c.JSON(200, events)
}
