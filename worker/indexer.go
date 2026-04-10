package worker

import (
	"analytics-backend/database"
	"analytics-backend/metrics"
	"analytics-backend/models"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type IndexStore interface {
	ReadIndexJobs() ([]redis.XMessage, error)
	BulkIndexEvents(events []models.Event) error
	AckIndexJobs(ids ...string) error
}

type DefaultIndexStore struct {
	Consumer string
}

func (s *DefaultIndexStore) ReadIndexJobs() ([]redis.XMessage, error) {
	return database.ReadIndexJobsFromGroup(s.Consumer)
}

func (s *DefaultIndexStore) BulkIndexEvents(events []models.Event) error {
	return database.BulkIndexEvents(database.Ctx, events)
}

func (s *DefaultIndexStore) AckIndexJobs(ids ...string) error {
	return database.AckIndexJobs(ids...)
}

func StartSearchIndexerWorker(workerName string, store IndexStore) {
	log.Printf("Starting search indexer worker %s...", workerName)
	metrics.ActiveWorkers.WithLabelValues("indexer").Inc()
	defer metrics.ActiveWorkers.WithLabelValues("indexer").Dec()

	for {
		metrics.WorkerIterations.WithLabelValues("indexer").Inc()
		if err := processIndexBatch(store); err != nil {
			metrics.SearchIndexFailures.WithLabelValues("batch").Inc()
			log.Printf("Error processing index batch for %s: %v", workerName, err)
			time.Sleep(time.Second)
			continue
		}
	}
}

func processIndexBatch(store IndexStore) error {
	started := time.Now()
	messages, err := store.ReadIndexJobs()
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}
	metrics.SearchIndexBatchSize.Observe(float64(len(messages)))

	events := make([]models.Event, 0, len(messages))
	ackIDs := make([]string, 0, len(messages))

	for _, msg := range messages {
		event, err := parseIndexEvent(msg)
		if err != nil {
			metrics.SearchIndexFailures.WithLabelValues("parse").Inc()
			log.Printf("Dropping malformed index message %s: %v", msg.ID, err)
			ackIDs = append(ackIDs, msg.ID)
			continue
		}

		events = append(events, event)
		ackIDs = append(ackIDs, msg.ID)
	}

	if len(events) > 0 {
		if err := store.BulkIndexEvents(events); err != nil {
			metrics.SearchIndexFailures.WithLabelValues("bulk_index").Inc()
			return err
		}
		metrics.SearchEventsIndexed.Add(float64(len(events)))
	}

	if err := store.AckIndexJobs(ackIDs...); err != nil {
		metrics.SearchIndexFailures.WithLabelValues("ack").Inc()
		return err
	}

	metrics.SearchIndexDuration.Observe(time.Since(started).Seconds())
	return nil
}

func parseIndexEvent(msg redis.XMessage) (models.Event, error) {
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

	timestampString, ok := values["timestamp"].(string)
	if !ok {
		return models.Event{}, strconv.ErrSyntax
	}

	timestamp, err := time.Parse(time.RFC3339Nano, timestampString)
	if err != nil {
		return models.Event{}, err
	}

	var id int64
	switch rawID := values["id"].(type) {
	case string:
		id, err = strconv.ParseInt(rawID, 10, 64)
	case int64:
		id = rawID
	case float64:
		id = int64(rawID)
	default:
		err = strconv.ErrSyntax
	}
	if err != nil {
		return models.Event{}, err
	}

	return models.Event{
		ID:        id,
		UserId:    userID,
		Action:    action,
		Element:   element,
		Duration:  duration,
		Timestamp: timestamp,
	}, nil
}
