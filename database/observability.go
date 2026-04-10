package database

import (
	"analytics-backend/metrics"
	"context"
	"log"
	"time"
)

type monitoredStream struct {
	stream string
	group  string
}

var monitoredStreams = []monitoredStream{
	{stream: StreamName, group: GroupName},
	{stream: IndexStreamName, group: IndexGroupName},
}

func observeDBOperation(backend, operation, target string, started time.Time, err error) {
	metrics.DBQueryDuration.WithLabelValues(backend, operation, target).Observe(time.Since(started).Seconds())
	if err != nil {
		metrics.DBOperationFailures.WithLabelValues(backend, operation, target).Inc()
	}
}

func observeRedisOperation(operation, stream string, started time.Time, err error) {
	metrics.RedisOperationDuration.WithLabelValues(operation, stream).Observe(time.Since(started).Seconds())
	if err != nil {
		metrics.RedisOperationFailures.WithLabelValues(operation, stream).Inc()
	}
}

func StartMetricsCollector(ctx context.Context) {
	updateMetricsSnapshot(ctx)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updateMetricsSnapshot(ctx)
		}
	}
}

func updateMetricsSnapshot(ctx context.Context) {
	collectPostgresConnections()
	collectRedisConnections()
	collectStreamMetrics(ctx)
}

func collectPostgresConnections() {
	if DB == nil {
		return
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Failed to access Postgres stats: %v", err)
		return
	}

	stats := sqlDB.Stats()
	metrics.DBConnections.WithLabelValues("postgres", "open").Set(float64(stats.OpenConnections))
	metrics.DBConnections.WithLabelValues("postgres", "in_use").Set(float64(stats.InUse))
	metrics.DBConnections.WithLabelValues("postgres", "idle").Set(float64(stats.Idle))
}

func collectRedisConnections() {
	if Rdb == nil {
		return
	}

	stats := Rdb.PoolStats()
	metrics.DBConnections.WithLabelValues("redis", "open").Set(float64(stats.TotalConns))
	metrics.DBConnections.WithLabelValues("redis", "in_use").Set(float64(stats.TotalConns - stats.IdleConns))
	metrics.DBConnections.WithLabelValues("redis", "idle").Set(float64(stats.IdleConns))
}

func collectStreamMetrics(ctx context.Context) {
	if Rdb == nil {
		return
	}

	for _, stream := range monitoredStreams {
		length, err := Rdb.XLen(ctx, stream.stream).Result()
		if err == nil {
			metrics.StreamLength.WithLabelValues(stream.stream, stream.group).Set(float64(length))
		} else {
			log.Printf("Failed to collect stream length for %s: %v", stream.stream, err)
		}

		pending, err := Rdb.XPending(ctx, stream.stream, stream.group).Result()
		if err == nil {
			metrics.StreamBacklog.WithLabelValues(stream.stream, stream.group).Set(float64(pending.Count))
		} else {
			log.Printf("Failed to collect stream backlog for %s/%s: %v", stream.stream, stream.group, err)
		}

		consumers, err := Rdb.XInfoConsumers(ctx, stream.stream, stream.group).Result()
		if err != nil {
			log.Printf("Failed to collect consumer info for %s/%s: %v", stream.stream, stream.group, err)
			continue
		}

		for _, consumer := range consumers {
			metrics.StreamConsumerLag.WithLabelValues(
				stream.stream,
				stream.group,
				consumer.Name,
			).Set(float64(consumer.Pending))
		}
	}
}
