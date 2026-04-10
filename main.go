package main

import (
	"analytics-backend/config"
	"analytics-backend/database"
	"analytics-backend/handlers"
	"analytics-backend/metrics"
	"analytics-backend/utils"
	"analytics-backend/worker"
	"context"
	"fmt"
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
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database.InitRedis(cfg.Redis)
	database.Initdb(cfg.Postgres)
	database.InitClickHouse(cfg.ClickHouse)
	if err := database.InitElasticsearch(cfg.Elasticsearch); err != nil {
		log.Fatalf("Failed to initialize Elasticsearch: %v", err)
	}
	utils.InitSnowflake(1)

	if err := database.EnsureConsumerGroup(); err != nil {
		log.Fatalf("Failed to create consumer group: %v", err)
	}
	if err := database.EnsureIndexerGroup(); err != nil {
		log.Fatalf("Failed to create indexer group: %v", err)
	}
	log.Println("Consumer group created successfully")
	go database.StartMetricsCollector(ctx)

	log.Println("Starting 4 background workers...")
	for i := 0; i < 4; i++ {
		workerName := fmt.Sprintf("worker-%d", i+1)
		eventStore := &worker.DefaultEventStore{Consumer: workerName}
		go worker.StartAggregatorWorker(workerName, eventStore)
	}
	log.Println("Starting 2 search indexer workers...")
	for i := 0; i < 2; i++ {
		workerName := fmt.Sprintf("indexer-%d", i+1)
		indexStore := &worker.DefaultIndexStore{Consumer: workerName}
		go worker.StartSearchIndexerWorker(workerName, indexStore)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.Use(metrics.PrometheusMiddleware())

	router.MaxMultipartMemory = 32 << 20

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.POST("/event", handlers.GetEvent)
	router.GET("/events", handlers.FetchEvents)
	router.GET("/events/recent", handlers.GetRecentFeed)
	router.GET("/events/stream", handlers.GetEventsStream)
	router.GET("/search/events", handlers.SearchEvents)
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
