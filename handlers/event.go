package handlers

import (
	"analytics-backend/database"
	"analytics-backend/metrics"
	"analytics-backend/models"
	"analytics-backend/utils"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

func GetEvent(c *gin.Context) {
	metrics.EventsReceived.Inc()

	var event models.Event
	if err := c.ShouldBind(&event); err != nil {
		metrics.EventsFailed.WithLabelValues("parse").Inc()
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Generate Snowflake ID
	event.ID = utils.GenerateID()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := database.AddToStreamWithContext(ctx, event); err != nil {
		metrics.EventsFailed.WithLabelValues("ingest").Inc()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	metrics.EventsIngested.Inc()
	c.JSON(202, gin.H{"status": "accepted", "event": event})
}
