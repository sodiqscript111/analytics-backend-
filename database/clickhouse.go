package database

import (
	"analytics-backend/config"
	"analytics-backend/models"
	"context"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var CH driver.Conn

func InitClickHouse(cfg config.ClickHouseConfig) {
	var err error
	CH, err = clickhouse.Open(&clickhouse.Options{
		Addr: cfg.Addr,
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Debug: true,
	})

	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}

	if err := CH.Ping(context.Background()); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			log.Fatalf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		log.Fatalf("Failed to ping ClickHouse: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS events (
		user_id String,
		action String,
		element String,
		duration Float64,
		timestamp DateTime
	) ENGINE = MergeTree()
	ORDER BY (action, timestamp)
	PARTITION BY toYYYYMM(timestamp)
	TTL timestamp + INTERVAL 1 MONTH
	`

	if err := CH.Exec(context.Background(), schema); err != nil {
		log.Fatalf("Failed to create ClickHouse table: %v", err)
	}

	log.Println("Connected to ClickHouse and ensured schema exists")
}

func InsertToClickHouse(ctx context.Context, userID, action, element string, duration float64, timestamp time.Time) error {
	started := time.Now()
	batch, err := CH.PrepareBatch(ctx, "INSERT INTO events")
	if err != nil {
		observeDBOperation("clickhouse", "prepare_batch", "events", started, err)
		return err
	}

	if err := batch.Append(userID, action, element, duration, timestamp); err != nil {
		observeDBOperation("clickhouse", "append", "events", started, err)
		return err
	}

	err = batch.Send()
	observeDBOperation("clickhouse", "insert", "events", started, err)
	return err
}

func BatchInsertToClickHouse(events []models.Event) error {
	if len(events) == 0 {
		return nil
	}

	started := time.Now()
	ctx := context.Background()
	batch, err := CH.PrepareBatch(ctx, "INSERT INTO events")
	if err != nil {
		observeDBOperation("clickhouse", "prepare_batch", "events", started, err)
		return err
	}

	for _, e := range events {
		if err := batch.Append(e.UserId, e.Action, e.Element, e.Duration, e.Timestamp); err != nil {
			observeDBOperation("clickhouse", "append", "events", started, err)
			return err
		}
	}

	err = batch.Send()
	observeDBOperation("clickhouse", "batch_insert", "events", started, err)
	return err
}

type ClickHouseAnalytics struct {
	Action      string  `ch:"action"`
	Count       uint64  `ch:"count"`
	AvgDuration float64 `ch:"avg_duration"`
}

func GetAnalyticsFromClickHouse(ctx context.Context) ([]ClickHouseAnalytics, error) {
	started := time.Now()
	var results []ClickHouseAnalytics
	query := `
		SELECT 
			action, 
			count(*) as count, 
			avg(duration) as avg_duration 
		FROM events 
		GROUP BY action
		ORDER BY count DESC
	`
	if err := CH.Select(ctx, &results, query); err != nil {
		observeDBOperation("clickhouse", "select", "events", started, err)
		return nil, err
	}
	observeDBOperation("clickhouse", "select", "events", started, nil)
	return results, nil
}
