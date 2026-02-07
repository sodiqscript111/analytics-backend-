package handlers

import (
	"analytics-backend/database"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAnalyticsClickHouse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	results, err := database.GetAnalyticsFromClickHouse(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Transform for frontend consistency if needed, or return direct
	response := make(map[string]interface{})
	actionCounts := make(map[string]uint64)
	var totalDuration float64
	var totalEvents uint64

	for _, r := range results {
		actionCounts[r.Action] = r.Count
		totalDuration += r.AvgDuration * float64(r.Count)
		totalEvents += r.Count
	}

	var avgDuration float64
	if totalEvents > 0 {
		avgDuration = totalDuration / float64(totalEvents)
	}

	response["action_counts"] = actionCounts
	response["avg_duration"] = avgDuration
	response["total_events"] = totalEvents
	response["processing_type"] = "clickhouse"

	c.JSON(200, response)
}
