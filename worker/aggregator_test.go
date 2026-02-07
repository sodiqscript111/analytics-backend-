package worker

import (
	"analytics-backend/models"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockEventStore implements EventStore for testing
type MockEventStore struct {
	ReadFromGroupFunc               func() ([]redis.XMessage, error)
	BatchCreateAggregatedEventsFunc func(aggEvents []*models.AggregatedEvent) error
	BatchInsertToClickHouseFunc     func(events []models.Event) error
	BatchCreateUserEventMapsFunc    func(userMaps []models.UserEventMap) error
	PushToRecentFeedFunc            func(ctx context.Context, data []byte, id int64) error
	AckMessageFunc                  func(ids ...string) error
}

func (m *MockEventStore) ReadFromGroup() ([]redis.XMessage, error) {
	if m.ReadFromGroupFunc != nil {
		return m.ReadFromGroupFunc()
	}
	return nil, nil
}

func (m *MockEventStore) BatchCreateAggregatedEvents(aggEvents []*models.AggregatedEvent) error {
	if m.BatchCreateAggregatedEventsFunc != nil {
		return m.BatchCreateAggregatedEventsFunc(aggEvents)
	}
	return nil
}

func (m *MockEventStore) BatchInsertToClickHouse(events []models.Event) error {
	if m.BatchInsertToClickHouseFunc != nil {
		return m.BatchInsertToClickHouseFunc(events)
	}
	return nil
}

func (m *MockEventStore) BatchCreateUserEventMaps(userMaps []models.UserEventMap) error {
	if m.BatchCreateUserEventMapsFunc != nil {
		return m.BatchCreateUserEventMapsFunc(userMaps)
	}
	return nil
}

func (m *MockEventStore) PushToRecentFeed(ctx context.Context, data []byte, id int64) error {
	if m.PushToRecentFeedFunc != nil {
		return m.PushToRecentFeedFunc(ctx, data, id)
	}
	return nil
}

func (m *MockEventStore) AckMessage(ids ...string) error {
	if m.AckMessageFunc != nil {
		return m.AckMessageFunc(ids...)
	}
	return nil
}

func TestProcessAggregatedBatch_Success(t *testing.T) {
	mockStore := &MockEventStore{
		ReadFromGroupFunc: func() ([]redis.XMessage, error) {
			return []redis.XMessage{
				{
					ID: "1-0",
					Values: map[string]interface{}{
						"user_id":   "user1",
						"action":    "click",
						"element":   "button1",
						"timestamp": time.Now().Format(time.RFC3339),
					},
				},
				{
					ID: "2-0",
					Values: map[string]interface{}{
						"user_id":   "user2",
						"action":    "click",
						"element":   "button1",
						"timestamp": time.Now().Format(time.RFC3339),
					},
				},
			}, nil
		},
		BatchCreateAggregatedEventsFunc: func(aggEvents []*models.AggregatedEvent) error {
			if len(aggEvents) != 1 {
				t.Errorf("Expected 1 aggregated event, got %d", len(aggEvents))
			}
			if aggEvents[0].Count != 2 {
				t.Errorf("Expected count 2, got %d", aggEvents[0].Count)
			}
			// Simulate DB assigning ID
			aggEvents[0].ID = 100
			return nil
		},
		BatchCreateUserEventMapsFunc: func(userMaps []models.UserEventMap) error {
			if len(userMaps) != 2 {
				t.Errorf("Expected 2 user maps, got %d", len(userMaps))
			}
			if userMaps[0].AggregatedEventID != 100 {
				t.Errorf("Expected AggregatedEventID 100, got %d", userMaps[0].AggregatedEventID)
			}
			return nil
		},
		AckMessageFunc: func(ids ...string) error {
			if len(ids) != 2 {
				t.Errorf("Expected 2 acked messages, got %d", len(ids))
			}
			return nil
		},
	}

	err := processAggregatedBatch(mockStore)
	if err != nil {
		t.Errorf("processAggregatedBatch failed: %v", err)
	}
}

func TestProcessAggregatedBatch_Empty(t *testing.T) {
	mockStore := &MockEventStore{
		ReadFromGroupFunc: func() ([]redis.XMessage, error) {
			return []redis.XMessage{}, nil
		},
	}

	err := processAggregatedBatch(mockStore)
	if err != nil {
		t.Errorf("Expected no error for empty batch, got %v", err)
	}
}

func TestProcessAggregatedBatch_ReadError(t *testing.T) {
	mockStore := &MockEventStore{
		ReadFromGroupFunc: func() ([]redis.XMessage, error) {
			return nil, errors.New("redis error")
		},
	}

	err := processAggregatedBatch(mockStore)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
