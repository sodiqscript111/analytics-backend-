package worker

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"log"
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

		time.Sleep(100 * time.Millisecond)
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

	// Group events by action + element + time window
	eventGroups := make(map[AggregationKey]*AggregatedData)
	var messageIDs []string

	for _, msg := range result {
		event := parseEvent(msg)

		// Round timestamp to aggregation window
		window := event.Timestamp.Truncate(AggregationWindow)

		// Create key WITHOUT user_id (aggregate across users)
		key := AggregationKey{
			Action:  event.Action,
			Element: event.Element,
			Window:  window.Format(time.RFC3339),
		}

		if existing, found := eventGroups[key]; found {
			// Add user to existing group
			existing.UserIDs = append(existing.UserIDs, event.UserId)
		} else {
			// Create new group
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

	for _, data := range eventGroups {

		aggEvent := models.AggregatedEvent{
			Action:  data.Action,
			Element: data.Element,
			Count:   len(data.UserIDs),
			Window:  data.Window,
		}

		if err := database.CreateAggregatedEvent(&aggEvent); err != nil {
			log.Printf("Failed to create aggregated event: %v", err)
			return err
		}

		var userMaps []models.UserEventMap
		for _, userID := range data.UserIDs {
			userMaps = append(userMaps, models.UserEventMap{
				AggregatedEventID: aggEvent.ID,
				UserID:            userID,
			})
		}

		if err := database.BatchCreateUserEventMaps(userMaps); err != nil {
			log.Printf("Failed to create user event maps: %v", err)
			return err
		}
	}

	if err := database.AckMessage(messageIDs...); err != nil {
		log.Printf("Failed to ack messages: %v", err)
		return err
	}

	log.Printf("Successfully processed and aggregated %d events", len(result))
	return nil
}
