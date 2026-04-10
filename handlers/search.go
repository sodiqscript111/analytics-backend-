package handlers

import (
	"analytics-backend/database"
	"analytics-backend/metrics"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func SearchEvents(c *gin.Context) {
	started := time.Now()
	status := "success"
	source := "elasticsearch"
	defer func() {
		metrics.SearchQueries.WithLabelValues(status, source).Inc()
		metrics.SearchQueryDuration.WithLabelValues(status, source).Observe(time.Since(started).Seconds())
	}()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	params := database.SearchEventsParams{
		Query:  strings.TrimSpace(c.Query("q")),
		Action: strings.TrimSpace(c.Query("action")),
		UserID: strings.TrimSpace(c.Query("user_id")),
		Size:   size,
		Cursor: strings.TrimSpace(c.Query("cursor")),
	}

	if from := strings.TrimSpace(c.Query("from")); from != "" {
		parsed, err := parseSearchTime(from)
		if err != nil {
			status = "invalid_request"
			c.JSON(400, gin.H{"error": "invalid from timestamp"})
			return
		}
		params.From = &parsed
	}

	if to := strings.TrimSpace(c.Query("to")); to != "" {
		parsed, err := parseSearchTime(to)
		if err != nil {
			status = "invalid_request"
			c.JSON(400, gin.H{"error": "invalid to timestamp"})
			return
		}
		params.To = &parsed
	}

	results, err := database.SearchEvents(ctx, params)
	if err != nil {
		status = "error"
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if results != nil && results.Source != "" {
		source = results.Source
	}

	c.JSON(200, results)
}

func parseSearchTime(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", raw)
}
