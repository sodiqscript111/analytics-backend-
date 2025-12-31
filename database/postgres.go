package database

import (
	"analytics-backend/models"
	"context"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Initdb() {
	dsn := "host=db user=postgres password=password dbname=testing port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Printf("Failed to connect to db:5432, trying localhost:5432")
		dsn = "host=localhost user=postgres password=password dbname=testing port=5432 sslmode=disable"
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

		if err != nil {
			log.Fatalf("Failed to connect to Postgres: %v", err)
		}
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
	return DB.Create(&event).Error
}

func AddToDatabaseWithContext(ctx context.Context, event models.Event) error {
	return DB.WithContext(ctx).Create(&event).Error
}

func BatchAddToDatabase(events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	return DB.CreateInBatches(events, 100).Error
}

func BatchAddToDatabaseWithContext(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	return DB.WithContext(ctx).CreateInBatches(events, 100).Error
}

func CreateAggregatedEvent(aggEvent *models.AggregatedEvent) error {
	return DB.Create(aggEvent).Error
}

func BatchCreateAggregatedEvents(aggEvents []*models.AggregatedEvent) error {
	if len(aggEvents) == 0 {
		return nil
	}
	return DB.CreateInBatches(aggEvents, 100).Error
}

func BatchCreateUserEventMaps(userMaps []models.UserEventMap) error {
	if len(userMaps) == 0 {
		return nil
	}
	return DB.CreateInBatches(userMaps, 500).Error
}

func GetEvents(limit int) ([]models.Event, error) {
	var events []models.Event
	result := DB.Limit(limit).Order("timestamp desc").Find(&events)
	return events, result.Error
}

func GetEventsWithContext(ctx context.Context, limit int) ([]models.Event, error) {
	var events []models.Event
	result := DB.WithContext(ctx).Limit(limit).Order("timestamp desc").Find(&events)
	return events, result.Error
}

func GetAggregatedEvents(limit int) ([]models.AggregatedEvent, error) {
	var events []models.AggregatedEvent
	result := DB.Limit(limit).Order("window desc").Find(&events)
	return events, result.Error
}

func GetUserEvents(userID string) ([]models.AggregatedEvent, error) {
	var events []models.AggregatedEvent
	result := DB.
		Joins("JOIN user_event_maps ON user_event_maps.aggregated_event_id = aggregated_events.id").
		Where("user_event_maps.user_id = ?", userID).
		Find(&events)
	return events, result.Error
}
