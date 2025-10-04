package database

import (
	"analytics-backend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func Initdb() {
	dsn := "host=localhost user=postgres password=password dbname=testing port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	log.Println("Connected to Postgres")

	db.AutoMigrate(&models.Event{})

	DB = db
}

func AddToDatabase(event models.Event) error {
	return DB.Create(&event).Error
}
