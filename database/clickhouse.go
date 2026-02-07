package database

import (
	"analytics-backend/models"
	"context"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var CH driver.Conn

func InitClickHouse() {
	var err error
	CH, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{"clickhouse:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		Debug: true,
	})

	if err != nil {
		// Fallback to localhost for local development
		log.Printf("Failed to connect to clickhouse:9000, trying localhost:9000")
		CH, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"localhost:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			Debug: true,
		})
		if err != nil {
			log.Fatalf("Failed to connect to ClickHouse: %v", err)
		}
	}

	if err := CH.Ping(context.Background()); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			log.Fatalf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		log.Fatalf("Failed to ping ClickHouse: %v", err)
	}

	// Create table
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
	batch, err := CH.PrepareBatch(ctx, "INSERT INTO events")
	if err != nil {
		return err
	}

	if err := batch.Append(userID, action, element, duration, timestamp); err != nil {
		return err
	}

	return batch.Send()
}

func BatchInsertToClickHouse(events []models.Event) error {
	if len(events) == 0 {
		return nil
	}

	ctx := context.Background()
	batch, err := CH.PrepareBatch(ctx, "INSERT INTO events")
	if err != nil {
		return err
	}

	for _, e := range events {
		if err := batch.Append(e.UserId, e.Action, e.Element, e.Duration, e.Timestamp); err != nil {
			return err
		}
	}

	return batch.Send()
}

type ClickHouseAnalytics struct {
	Action      string  `ch:"action"`
	Count       uint64  `ch:"count"`
	AvgDuration float64 `ch:"avg_duration"`
}

func GetAnalyticsFromClickHouse(ctx context.Context) ([]ClickHouseAnalytics, error) {
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
		return nil, err
	}
	return results, nil
}
