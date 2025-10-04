package database

import (
	"analytics-backend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

func Initdb() {
	dsn := "host=localhost user=postgres password=password dbname=testing port=5432 sslmode=disable "
	DB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	log.Println("Connected to Postgres")
	DB.AutoMigrate(&models.Event{})
}
