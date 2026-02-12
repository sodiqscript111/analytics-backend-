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

type EventStore interface {
	ReadFromGroup() ([]redis.XMessage, error)
	BatchCreateAggregatedEvents(aggEvents []*models.AggregatedEvent) error
	BatchInsertToClickHouse(events []models.Event) error
	BatchCreateUserEventMaps(userMaps []models.UserEventMap) error
	PushToRecentFeed(ctx context.Context, data []byte, id int64) error
	AckMessage(ids ...string) error
}

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

	metrics.AggregationBatchSize.Observe(float64(len(result)))
	log.Printf("Aggregating batch of %d events", len(result))

	eventGroups := make(map[AggregationKey]*AggregatedData)
	var messageIDs []string
	var decodedEvents []models.Event

	for _, msg := range result {
		event := parseEvent(msg)

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

	var allUserMaps []models.UserEventMap

	var aggEvents []*models.AggregatedEvent
	var userIDsList [][]string

	for _, data := range eventGroups {
		aggEvent := &models.AggregatedEvent{
			Action:  data.Action,
			Element: data.Element,
			Count:   len(data.UserIDs),
			Window:  data.Window,
		}
		aggEvents = append(aggEvents, aggEvent)
		userIDsList = append(userIDsList, data.UserIDs)
	}

	if err := store.BatchCreateAggregatedEvents(aggEvents); err != nil {
		log.Printf("Failed to batch create aggregated events: %v", err)
		return err
	}

	for i, aggEvent := range aggEvents {
		userIDs := userIDsList[i]
		for _, userID := range userIDs {
			allUserMaps = append(allUserMaps, models.UserEventMap{
				AggregatedEventID: aggEvent.ID,
				UserID:            userID,
			})
		}
	}

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

	for _, event := range decodedEvents {
		if jsonBytes, err := json.Marshal(event); err == nil {
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
