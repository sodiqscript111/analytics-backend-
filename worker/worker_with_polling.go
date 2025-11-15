package worker

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"github.com/redis/go-redis/v9"
	"log"
	"strconv"
	"time"
)

func StartWorker() {
	log.Println("Starting background worker...")

	for {
		if err := processBatch(); err != nil {
			log.Printf("Error processing batch: %v", err)
			time.Sleep(1 * time.Second) // Brief pause on error
			continue
		}

		// Small delay between batches to avoid hammering Redis
		time.Sleep(100 * time.Millisecond)
	}
}

func processBatch() error {
	result, err := database.ReadFromGroup()
	if err != nil {
		return err
	}

	if len(result) == 0 {
		return nil
	}

	log.Printf("Processing batch of %d events", len(result))

	var events []models.Event
	var messageIDs []string

	for _, msg := range result {
		event := parseEvent(msg)
		events = append(events, event)
		messageIDs = append(messageIDs, msg.ID)
	}

	if err := database.BatchAddToDatabase(events); err != nil {
		log.Printf("Failed to insert batch: %v", err)
		return err
	}

	if err := database.AckMessage(messageIDs...); err != nil {
		log.Printf("Failed to ack messages: %v", err)
		return err
	}

	log.Printf("Successfully processed %d events", len(events))
	return nil
}

func parseEvent(msg redis.XMessage) models.Event {
	var event models.Event

	if v, ok := msg.Values["user_id"].(string); ok {
		event.UserId = v
	}
	if v, ok := msg.Values["action"].(string); ok {
		event.Action = v
	}
	if v, ok := msg.Values["element"].(string); ok {
		event.Element = v
	}

	if durStr, ok := msg.Values["duration"].(string); ok {
		if dur, err := strconv.ParseFloat(durStr, 64); err == nil {
			event.Duration = dur
		}
	}

	if tsStr, ok := msg.Values["timestamp"].(string); ok {
		if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
			event.Timestamp = ts
		}
	}

	return event
}
