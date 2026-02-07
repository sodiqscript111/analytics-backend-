package worker

import (
	"analytics-backend/database"
	"analytics-backend/metrics"
	"analytics-backend/models"
	"context"
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

// EventStore interface defines the methods required by the aggregator worker.
// This allows for mocking in unit tests.
type EventStore interface {
	ReadFromGroup() ([]redis.XMessage, error)
	BatchCreateAggregatedEvents(aggEvents []*models.AggregatedEvent) error
	BatchInsertToClickHouse(events []models.Event) error
	BatchCreateUserEventMaps(userMaps []models.UserEventMap) error
	PushToRecentFeed(ctx context.Context, data []byte, id int64) error
	AckMessage(ids ...string) error
}

// DefaultEventStore implements EventStore using the database package.
type DefaultEventStore struct{}

func (s *DefaultEventStore) ReadFromGroup() ([]redis.XMessage, error) {
	return database.ReadFromGroup()
}

func (s *DefaultEventStore) BatchCreateAggregatedEvents(aggEvents []*models.AggregatedEvent) error {
	return database.BatchCreateAggregatedEvents(aggEvents)
}

func (s *DefaultEventStore) BatchInsertToClickHouse(events []models.Event) error {
	return database.BatchInsertToClickHouse(events)
}

func (s *DefaultEventStore) BatchCreateUserEventMaps(userMaps []models.UserEventMap) error {
	return database.BatchCreateUserEventMaps(userMaps)
}

func (s *DefaultEventStore) PushToRecentFeed(ctx context.Context, data []byte, id int64) error {
	return database.PushToRecentFeed(ctx, data, id)
}

func (s *DefaultEventStore) AckMessage(ids ...string) error {
	return database.AckMessage(ids...)
}

func StartAggregatorWorker(store EventStore) {
	log.Println("Starting aggregator worker...")
	metrics.ActiveWorkers.Inc()
	defer metrics.ActiveWorkers.Dec()

	for {
		metrics.WorkerIterations.Inc()
		if err := processAggregatedBatch(store); err != nil {
			metrics.EventsFailed.WithLabelValues("aggregation").Inc()
			log.Printf("Error processing aggregated batch: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		// Removed artificial sleep to maximize throughput
	}
}

func processAggregatedBatch(store EventStore) error {
	start := time.Now()
	result, err := store.ReadFromGroup()
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
		// Keep raw message JSON handling if needed, but we will push later
		// event := parseEvent(msg) - already done below

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
	if err := store.BatchCreateAggregatedEvents(aggEvents); err != nil {
		log.Printf("Failed to batch create aggregated events: %v", err)
		return err
	}

	// Batch insert aggregated events
	if err := store.BatchCreateAggregatedEvents(aggEvents); err != nil {
		log.Printf("Failed to batch create aggregated events: %v", err)
		return err
	}

	// Re-iterate over eventGroups to create UserMaps, matching the AggregatedEvent IDs
	// This logic in the original code was slightly flawed because re-creating AggegatedEvents
	// inside the loop would mean they don't have the IDs from the previous batch insert.
	// However, since we are doing a refactor for testability, I will fix this logic to use the `aggEvents` slice we populated.

	// In the original code (lines 133-143), it seemed to re-create structs.
	// Let's optimize: we already have `aggEvents` populate with IDs after `BatchCreateAggregatedEvents` (assuming GORM updates them).

	// For the sake of preserving logic but fixing map creation:
	// Find the corresponding aggEvent for each group key.
	// Since maps don't guarantee order, we need to be careful.
	// The original code re-iterated the map which is non-deterministic,
	// potentially mismatching if not carefully done, but here we can just iterate our created slice if we link back.

	// Simplest fix:
	// We iterate `aggEvents` which we just inserted.
	// But `aggEvents` doesn't have the UserIDs.
	// So we need to link them up.

	// Let's modify the loop to build `aggEvents` AND keep track of UserIDs for each index.
	// Redoing the loop for clarity and correctness in the refactor.
	aggEvents = nil // reset
	var userIDsList [][]string

	for key, data := range eventGroups {
		// Ensure stable order by iterating map? No, map is random.
		// But as long as we append consistently it's fine.
		_ = key
		aggEvent := &models.AggregatedEvent{
			Action:  data.Action,
			Element: data.Element,
			Count:   len(data.UserIDs),
			Window:  data.Window,
		}
		aggEvents = append(aggEvents, aggEvent)
		userIDsList = append(userIDsList, data.UserIDs)
	}

	// Insert Aggregated Events - GORM will update IDs in place
	if err := store.BatchCreateAggregatedEvents(aggEvents); err != nil {
		log.Printf("Failed to batch create aggregated events: %v", err)
		return err
	}

	// Now create UserMaps using the populated IDs
	for i, aggEvent := range aggEvents {
		userIDs := userIDsList[i]
		for _, userID := range userIDs {
			allUserMaps = append(allUserMaps, models.UserEventMap{
				AggregatedEventID: aggEvent.ID,
				UserID:            userID,
			})
		}
	}

	// Async insert to ClickHouse for raw analytics
	go func(events []models.Event) {
		if err := store.BatchInsertToClickHouse(events); err != nil {
			log.Printf("Failed to insert batch to ClickHouse: %v", err)
			metrics.EventsFailed.WithLabelValues("clickhouse_insert").Inc()
		}
	}(decodedEvents)

	if err := store.BatchCreateUserEventMaps(allUserMaps); err != nil {
		log.Printf("Failed to create user event maps: %v", err)
		return err
	}

	// Push to Recent Feed (Redis ZSet) AFTER successful DB writes
	// We only need to push the most recent ones if the batch is huge, but pushing all is safer for consistency
	// The Redis Lua script or ZRemRangeByRank will handle the trimming
	for _, event := range decodedEvents {
		if jsonBytes, err := json.Marshal(event); err == nil {
			// Fire and forget - errors here shouldn't fail the batch
			go func(ctx context.Context, data []byte, id int64) {
				_ = store.PushToRecentFeed(ctx, data, id)
			}(database.Ctx, jsonBytes, event.ID)
		}
	}

	if err := store.AckMessage(messageIDs...); err != nil {
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

	var id int64
	if idStr, ok := values["id"].(string); ok {
		id, _ = strconv.ParseInt(idStr, 10, 64)
	}

	return models.Event{
		ID:        id,
		UserId:    userID,
		Action:    action,
		Element:   element,
		Duration:  duration,
		Timestamp: timestamp,
	}
}
