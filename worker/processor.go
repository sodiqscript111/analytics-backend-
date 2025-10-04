package worker

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"strconv"
	"time"
)

func AddToDatabase() {
	// Read from the consumer group
	result, err := database.ReadFromGroup()
	if err != nil {
		panic(err)
	}

	for _, msg := range result {
		var event models.Event

		// Map Redis values into the Event struct
		if v, ok := msg.Values["user_id"].(string); ok {
			event.UserId = v
		}
		if v, ok := msg.Values["action"].(string); ok {
			event.Action = v
		}
		if v, ok := msg.Values["element"].(string); ok {
			event.Element = v
		}

		// Convert duration string → float64
		if durStr, ok := msg.Values["duration"].(string); ok {
			if dur, err := strconv.ParseFloat(durStr, 64); err == nil {
				event.Duration = dur
			}
		}

		// Convert timestamp string → time.Time
		if tsStr, ok := msg.Values["timestamp"].(string); ok {
			if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
				event.Timestamp = ts
			}
		}

		// Save into DB
		if err := database.AddToDatabase(event); err != nil {
			panic(err)
		}

		// Ack the message so it doesn’t pile up
		if err := database.AckMessage(msg.ID); err != nil {
			panic(err)
		}
	}
}
