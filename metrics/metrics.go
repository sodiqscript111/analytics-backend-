package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
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

	AggregationBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "analytics_aggregation_batch_size",
		Help:    "Number of events per aggregation batch",
		Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500, 5000},
	})

	AggregatedEventsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analytics_aggregated_events_created_total",
		Help: "Total number of aggregated event records created",
	})

	SearchQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "analytics_search_query_duration_seconds",
		Help:    "Search endpoint duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"status", "source"})

	SearchQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "analytics_search_queries_total",
		Help: "Total number of search queries served",
	}, []string{"status", "source"})

	SearchIndexBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "analytics_search_index_batch_size",
		Help:    "Number of events per search indexing batch",
		Buckets: []float64{10, 25, 50, 100, 200, 500, 1000},
	})

	SearchIndexDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "analytics_search_index_duration_seconds",
		Help:    "Time spent indexing event batches into Elasticsearch",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	})

	SearchEventsIndexed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analytics_search_events_indexed_total",
		Help: "Total number of events successfully indexed into Elasticsearch",
	})

	SearchIndexFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "analytics_search_index_failures_total",
		Help: "Total number of search indexing failures",
	}, []string{"stage"})

	SSESubscribers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "analytics_sse_subscribers",
		Help: "Number of active Server-Sent Events subscribers",
	})

	StreamLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_stream_length",
		Help: "Current Redis stream length",
	}, []string{"stream", "group"})

	StreamBacklog = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_stream_backlog",
		Help: "Number of pending messages in a Redis consumer group",
	}, []string{"stream", "group"})

	StreamConsumerLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_stream_consumer_pending",
		Help: "Pending messages assigned to a specific Redis stream consumer",
	}, []string{"stream", "group", "consumer"})

	RedisOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "analytics_redis_operation_duration_seconds",
		Help:    "Redis operation duration in seconds",
		Buckets: []float64{0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"operation", "stream"})

	RedisOperationFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "analytics_redis_operation_failures_total",
		Help: "Total number of failed Redis operations",
	}, []string{"operation", "stream"})

	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "analytics_db_query_duration_seconds",
		Help:    "Database query duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"backend", "operation", "target"})

	DBOperationFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "analytics_db_operation_failures_total",
		Help: "Total number of failed database operations",
	}, []string{"backend", "operation", "target"})

	DBConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_db_connections",
		Help: "Number of open connections per backend state",
	}, []string{"backend", "state"})

	ActiveWorkers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_active_workers",
		Help: "Number of currently active worker goroutines by type",
	}, []string{"worker_type"})

	WorkerIterations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "analytics_worker_iterations_total",
		Help: "Total number of worker loop iterations by type",
	}, []string{"worker_type"})
)
