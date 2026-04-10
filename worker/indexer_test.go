package worker

import (
	"analytics-backend/models"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type MockIndexStore struct {
	ReadIndexJobsFunc   func() ([]redis.XMessage, error)
	BulkIndexEventsFunc func(events []models.Event) error
	AckIndexJobsFunc    func(ids ...string) error
}

func (m *MockIndexStore) ReadIndexJobs() ([]redis.XMessage, error) {
	if m.ReadIndexJobsFunc != nil {
		return m.ReadIndexJobsFunc()
	}
	return nil, nil
}

func (m *MockIndexStore) BulkIndexEvents(events []models.Event) error {
	if m.BulkIndexEventsFunc != nil {
		return m.BulkIndexEventsFunc(events)
	}
	return nil
}

func (m *MockIndexStore) AckIndexJobs(ids ...string) error {
	if m.AckIndexJobsFunc != nil {
		return m.AckIndexJobsFunc(ids...)
	}
	return nil
}

func TestProcessIndexBatch_AcksOnSuccess(t *testing.T) {
	acked := false
	store := &MockIndexStore{
		ReadIndexJobsFunc: func() ([]redis.XMessage, error) {
			return []redis.XMessage{
				{
					ID: "1-0",
					Values: map[string]any{
						"id":        "123",
						"user_id":   "user-1",
						"action":    "click",
						"element":   "button",
						"duration":  "12.5",
						"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
					},
				},
			}, nil
		},
		BulkIndexEventsFunc: func(events []models.Event) error {
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			if events[0].ID != 123 {
				t.Fatalf("expected event id 123, got %d", events[0].ID)
			}
			return nil
		},
		AckIndexJobsFunc: func(ids ...string) error {
			acked = true
			if len(ids) != 1 {
				t.Fatalf("expected 1 ack id, got %d", len(ids))
			}
			return nil
		},
	}

	if err := processIndexBatch(store); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !acked {
		t.Fatal("expected index job to be acked")
	}
}

func TestProcessIndexBatch_DoesNotAckOnIndexFailure(t *testing.T) {
	acked := false
	store := &MockIndexStore{
		ReadIndexJobsFunc: func() ([]redis.XMessage, error) {
			return []redis.XMessage{
				{
					ID: "1-0",
					Values: map[string]any{
						"id":        "123",
						"user_id":   "user-1",
						"action":    "click",
						"element":   "button",
						"duration":  "12.5",
						"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
					},
				},
			}, nil
		},
		BulkIndexEventsFunc: func(events []models.Event) error {
			return errors.New("elastic down")
		},
		AckIndexJobsFunc: func(ids ...string) error {
			acked = true
			return nil
		},
	}

	if err := processIndexBatch(store); err == nil {
		t.Fatal("expected indexing error")
	}
	if acked {
		t.Fatal("expected index job to remain unacked on failure")
	}
}
