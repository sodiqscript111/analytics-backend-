package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Event ingestion metrics
	EventsReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analytics_events_received_total",
		Help: "Total number of events received via HTTP",
	})

	EventsIngested = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analytics_events_ingested_total",
		Help: "Total number of events successfully added to stream",
	})

	EventsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analytics_events_processed_total",
		Help: "Total number of events processed by workers",
	})

	EventsFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "analytics_events_failed_total",
		Help: "Total number of failed event operations",
	}, []string{"operation"})

	// Processing duration metrics
	EventProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "analytics_event_processing_seconds",
		Help:    "Time spent processing event batches",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "analytics_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "endpoint", "status"})

	// Aggregation metrics
	AggregationBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "analytics_aggregation_batch_size",
		Help:    "Number of events per aggregation batch",
		Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500, 5000},
	})

	AggregatedEventsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analytics_aggregated_events_created_total",
		Help: "Total number of aggregated event records created",
	})

	// Stream metrics
	StreamBacklog = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "analytics_stream_backlog",
		Help: "Number of unprocessed messages in Redis stream",
	})

	StreamConsumerLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_stream_consumer_lag",
		Help: "Consumer lag behind the stream",
	}, []string{"consumer"})

	// Database metrics
	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "analytics_db_query_duration_seconds",
		Help:    "Database query duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"operation", "table"})

	DBConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_db_connections",
		Help: "Number of database connections",
	}, []string{"state"})

	// Worker metrics
	ActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "analytics_active_workers",
		Help: "Number of currently active worker goroutines",
	})

	WorkerIterations = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analytics_worker_iterations_total",
		Help: "Total number of worker loop iterations",
	})
)
