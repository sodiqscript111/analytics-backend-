package main

import (
	"analytics-backend/config"
	"analytics-backend/database"
	"analytics-backend/handlers"
	"analytics-backend/metrics"
	"analytics-backend/utils"
	"analytics-backend/worker"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var ctx = context.Background()

func main() {
	// Load Configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database.InitRedis(cfg.Redis)
	database.Initdb(cfg.Postgres)
	database.InitClickHouse(cfg.ClickHouse)
	utils.InitSnowflake(1)

	if err := database.EnsureConsumerGroup(); err != nil {
		log.Fatalf("Failed to create consumer group: %v", err)
	}
	log.Println("Consumer group created successfully")

	log.Println("Starting 4 background workers...")
	eventStore := &worker.DefaultEventStore{}
	for i := 0; i < 4; i++ {
		go worker.StartAggregatorWorker(eventStore)
	}

	router := gin.Default()

	// Add Prometheus middleware
	router.Use(metrics.PrometheusMiddleware())

	router.MaxMultipartMemory = 32 << 20

	// Metrics endpoint for Prometheus scraping
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.POST("/event", handlers.GetEvent)
	router.GET("/events", handlers.FetchEvents)
	router.GET("/events/recent", handlers.GetRecentFeed)
	router.GET("/analytics/clickhouse", handlers.GetAnalyticsClickHouse)
	router.GET("/analytics/sequential", handlers.GetAnalyticsSequential)
	router.GET("/analytics/mapreduce", handlers.GetAnalyticsMapReduce)

	srv := &http.Server{
		Addr:           ":8080",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Println("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
