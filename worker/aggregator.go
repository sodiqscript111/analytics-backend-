package worker

import (
	"analytics-backend/database"
	"analytics-backend/metrics"
	"analytics-backend/models"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
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
	metrics.ActiveWorkers.Inc()
	defer metrics.ActiveWorkers.Dec()

	for {
		metrics.WorkerIterations.Inc()
		if err := processAggregatedBatch(); err != nil {
			metrics.EventsFailed.WithLabelValues("aggregation").Inc()
			log.Printf("Error processing aggregated batch: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		// Removed artificial sleep to maximize throughput
	}
}

func processAggregatedBatch() error {
	start := time.Now()
	result, err := database.ReadFromGroup()
	if err != nil {
		return err
	}

	if len(result) == 0 {
		return nil
	}

	// Track batch size
	metrics.AggregationBatchSize.Observe(float64(len(result)))
	log.Printf("Aggregating batch of %d events", len(result))

	eventGroups := make(map[AggregationKey]*AggregatedData)
	var messageIDs []string
	var decodedEvents []models.Event

	for _, msg := range result {
		// Push raw message JSON to recent feed
		// We use the raw Values map or marshal the parsed event.
		// Since we have the parsed event, let's marshal that to have a clean structure.
		event := parseEvent(msg)

		// Serialize event for the feed
		if jsonBytes, err := json.Marshal(event); err == nil {
			database.PushToRecentFeed(database.Ctx, jsonBytes)
		}

		decodedEvents = append(decodedEvents, event)

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
		batchItems = append(batchItems, BatchItem{AggEvent: aggEvent, UserIDs: data.UserIDs})
		aggEvents = append(aggEvents, aggEvent)
	}

	// Async insert to ClickHouse for raw analytics
	go func(events []models.Event) {
		if err := database.BatchInsertToClickHouse(events); err != nil {
			log.Printf("Failed to insert batch to ClickHouse: %v", err)
			metrics.EventsFailed.WithLabelValues("clickhouse_insert").Inc()
		}
	}(decodedEvents)

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

	// Track processing metrics
	metrics.EventProcessingDuration.Observe(time.Since(start).Seconds())
	metrics.EventsProcessed.Add(float64(len(result)))
	metrics.AggregatedEventsCreated.Add(float64(len(aggEvents)))

	return nil
}

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
