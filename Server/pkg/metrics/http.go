package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPMetrics holds Prometheus metrics for HTTP endpoints.
type HTTPMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge
}

// NewHTTPMetrics registers and returns standard HTTP metrics.
func NewHTTPMetrics(namespace string) *HTTPMetrics {
	factory := promauto.With(Registry)
	return &HTTPMetrics{
		RequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		}, []string{"method", "path", "status"}),

		RequestDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "path"}),

		ActiveRequests: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "active_requests",
			Help:      "Number of in-flight HTTP requests",
		}),
	}
}

// GinMiddleware returns a Gin middleware that records HTTP metrics.
func (m *HTTPMetrics) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.ActiveRequests.Inc()
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		// Use the route template (e.g. "/rooms/:id") to avoid cardinality explosion.
		// c.FullPath() returns the registered route pattern; fall back for unmatched routes.
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		m.RequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		m.RequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
		m.ActiveRequests.Dec()
	}
}
