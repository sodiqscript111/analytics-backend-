package main

import (
	"analytics-backend/config"
	"analytics-backend/database"
	"context"
	"log"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database.Initdb(cfg.Postgres)
	if err := database.InitElasticsearch(cfg.Elasticsearch); err != nil {
		log.Fatalf("Failed to initialize Elasticsearch: %v", err)
	}

	if err := database.BackfillEventsToElasticsearch(context.Background(), 200); err != nil {
		log.Fatalf("Backfill failed: %v", err)
	}

	log.Println("Search backfill completed successfully")
}
