package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			status,
		).Observe(duration)
	}
}
