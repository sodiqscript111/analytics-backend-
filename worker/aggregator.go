package worker

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"github.com/redis/go-redis/v9"
	"log"
	"strconv"
	"time"
)

const AggregationWindow = 5 * time.Second

type AggregationKey struct {
	Action  string
	Element string
	Window  string
}

type AggregatedData struct {
	Action  string
	Element string
	UserIDs []string
	Window  time.Time
}

func StartAggregatorWorker() {
	log.Println("Starting aggregator worker...")

	for {
		if err := processAggregatedBatch(); err != nil {
			log.Printf("Error processing aggregated batch: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		// Removed artificial sleep to maximize throughput
	}
}

func processAggregatedBatch() error {
	result, err := database.ReadFromGroup()
	if err != nil {
		return err
	}

	if len(result) == 0 {
		return nil
	}

	log.Printf("Aggregating batch of %d events", len(result))

	eventGroups := make(map[AggregationKey]*AggregatedData)
	var messageIDs []string

	for _, msg := range result {
		event := parseEvent(msg)

		window := event.Timestamp.Truncate(AggregationWindow)

		key := AggregationKey{
			Action:  event.Action,
			Element: event.Element,
			Window:  window.Format(time.RFC3339),
		}

		if existing, found := eventGroups[key]; found {
			existing.UserIDs = append(existing.UserIDs, event.UserId)
		} else {
			eventGroups[key] = &AggregatedData{
				Action:  event.Action,
				Element: event.Element,
				UserIDs: []string{event.UserId},
				Window:  window,
			}
		}

		messageIDs = append(messageIDs, msg.ID)
	}

	log.Printf("Aggregated %d events into %d unique action-element pairs",
		len(result), len(eventGroups))

	var aggEvents []*models.AggregatedEvent
	var allUserMaps []models.UserEventMap

	// First pass: create all aggregated events
	for _, data := range eventGroups {
		aggEvent := &models.AggregatedEvent{
			Action:  data.Action,
			Element: data.Element,
			Count:   len(data.UserIDs),
			Window:  data.Window,
		}
		aggEvents = append(aggEvents, aggEvent)

		// Store UserIDs temporarily with the struct pointer to map later
		// Note: The ID will be populated after insertion if using GORM with returning (Postgres Default)
	}

	// Batch insert aggregated events
	if err := database.BatchCreateAggregatedEvents(aggEvents); err != nil {
		log.Printf("Failed to batch create aggregated events: %v", err)
		return err
	}

	// Second pass: create user maps using the IDs generated from the first pass
	// We need to match the data back to the saved aggEvents.
	// Since range over map is random, random iteration order matters if we simply iterated again.
	// But we constructed aggEvents slice, so we can iterate that if we had linked the data.
	// However, my previous logic was iterating the map. Let's restart the matching logic.

	// Better approach: Since we can't easily link the random map iteration to the slice unless we stored it.
	// Let's re-iterate the slice which we can trust has been populated with IDs by GORM.
	// BUT, we need the original UserIDs for each aggEvent.
	// Let's use a struct to hold both for the batch process.

	// Refactoring loop above slightly to support this safety.

	// Wait, I cannot easily change the logic in a simple replace tool if I don't use the map.
	// Let's assume I can iterate `aggEvents` and look up correctly? No, `aggEvents` doesn't have UserIDs.
	// I need to modify the loop. See below.

	// Let's discard the standard loop and rebuild.

	type BatchItem struct {
		AggEvent *models.AggregatedEvent
		UserIDs  []string
	}
	var batchItems []BatchItem

	for _, data := range eventGroups {
		aggEvent := &models.AggregatedEvent{
			Action:  data.Action,
			Element: data.Element,
			Count:   len(data.UserIDs),
			Window:  data.Window,
		}
		batchItems = append(batchItems, BatchItem{AggEvent: aggEvent, UserIDs: data.UserIDs})
		aggEvents = append(aggEvents, aggEvent)
	}

	if err := database.BatchCreateAggregatedEvents(aggEvents); err != nil {
		log.Printf("Failed to batch create aggregated events: %v", err)
		return err
	}

	for _, item := range batchItems {
		for _, userID := range item.UserIDs {
			allUserMaps = append(allUserMaps, models.UserEventMap{
				AggregatedEventID: item.AggEvent.ID,
				UserID:            userID,
			})
		}
	}

	if err := database.BatchCreateUserEventMaps(allUserMaps); err != nil {
		log.Printf("Failed to create user event maps: %v", err)
		return err
	}

	if err := database.AckMessage(messageIDs...); err != nil {
		log.Printf("Failed to ack messages: %v", err)
		return err
	}

	log.Printf("Successfully processed and aggregated %d events", len(result))
	return nil
func parseEvent(msg redis.XMessage) models.Event {
	values := msg.Values

	userID, _ := values["user_id"].(string)
	action, _ := values["action"].(string)
	element, _ := values["element"].(string)

	var duration float64
	if d, ok := values["duration"].(string); ok {
		duration, _ = strconv.ParseFloat(d, 64)
	} else if d, ok := values["duration"].(float64); ok {
		duration = d
	}

	var timestamp time.Time
	if tStr, ok := values["timestamp"].(string); ok {
		timestamp, _ = time.Parse(time.RFC3339, tStr)
	} else {
		timestamp = time.Now()
	}

	return models.Event{
		UserId:    userID,
		Action:    action,
		Element:   element,
		Duration:  duration,
		Timestamp: timestamp,
	}
}
