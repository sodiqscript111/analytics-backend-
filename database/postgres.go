package database

import (
	"analytics-backend/config"
	"analytics-backend/models"
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Initdb(cfg config.PostgresConfig) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get DB instance: %v", err)
	}

	sqlDB.SetMaxOpenConns(200)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	log.Println("Connected to Postgres with connection pool configured")

	if err := db.AutoMigrate(
		&models.Event{},
		&models.AggregatedEvent{},
		&models.UserEventMap{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	DB = db
}

func AddToDatabase(event models.Event) error {
	started := time.Now()
	err := DB.Create(&event).Error
	observeDBOperation("postgres", "create", "events", started, err)
	return err
}

func AddToDatabaseWithContext(ctx context.Context, event models.Event) error {
	started := time.Now()
	err := DB.WithContext(ctx).Create(&event).Error
	observeDBOperation("postgres", "create", "events", started, err)
	return err
}

func BatchAddToDatabase(events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	started := time.Now()
	err := DB.CreateInBatches(events, 100).Error
	observeDBOperation("postgres", "batch_create", "events", started, err)
	return err
}

func BatchAddToDatabaseWithContext(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	started := time.Now()
	err := DB.WithContext(ctx).CreateInBatches(events, 100).Error
	observeDBOperation("postgres", "batch_create", "events", started, err)
	return err
}

func CreateAggregatedEvent(aggEvent *models.AggregatedEvent) error {
	started := time.Now()
	err := DB.Create(aggEvent).Error
	observeDBOperation("postgres", "create", "aggregated_events", started, err)
	return err
}

func BatchCreateAggregatedEvents(aggEvents []*models.AggregatedEvent) error {
	if len(aggEvents) == 0 {
		return nil
	}
	started := time.Now()
	err := DB.CreateInBatches(aggEvents, 100).Error
	observeDBOperation("postgres", "batch_create", "aggregated_events", started, err)
	return err
}

func BatchCreateUserEventMaps(userMaps []models.UserEventMap) error {
	if len(userMaps) == 0 {
		return nil
	}
	started := time.Now()
	err := DB.CreateInBatches(userMaps, 500).Error
	observeDBOperation("postgres", "batch_create", "user_event_maps", started, err)
	return err
}

func GetEvents(limit int) ([]models.Event, error) {
	started := time.Now()
	var events []models.Event
	result := DB.Limit(limit).Order("timestamp desc").Find(&events)
	observeDBOperation("postgres", "select", "events", started, result.Error)
	return events, result.Error
}

func GetEventsWithContext(ctx context.Context, limit int) ([]models.Event, error) {
	started := time.Now()
	var events []models.Event
	result := DB.WithContext(ctx).Limit(limit).Order("timestamp desc").Find(&events)
	observeDBOperation("postgres", "select", "events", started, result.Error)
	return events, result.Error
}

func GetAggregatedEvents(limit int) ([]models.AggregatedEvent, error) {
	started := time.Now()
	var events []models.AggregatedEvent
	result := DB.Limit(limit).Order("window desc").Find(&events)
	observeDBOperation("postgres", "select", "aggregated_events", started, result.Error)
	return events, result.Error
}

func GetUserEvents(userID string) ([]models.AggregatedEvent, error) {
	started := time.Now()
	var events []models.AggregatedEvent
	result := DB.
		Joins("JOIN user_event_maps ON user_event_maps.aggregated_event_id = aggregated_events.id").
		Where("user_event_maps.user_id = ?", userID).
		Find(&events)
	observeDBOperation("postgres", "select_join", "user_event_maps", started, result.Error)
	return events, result.Error
}

func FindEventsInBatches(batchSize int, fn func([]models.Event) error) error {
	if batchSize <= 0 {
		batchSize = 100
	}

	var events []models.Event

	started := time.Now()
	err := DB.
		Model(&models.Event{}).
		Order("timestamp asc").
		FindInBatches(&events, batchSize, func(tx *gorm.DB, batch int) error {
			if len(events) == 0 {
				return nil
			}
			return fn(append([]models.Event(nil), events...))
		}).Error
	observeDBOperation("postgres", "find_in_batches", "events", started, err)
	return err
}
