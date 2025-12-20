package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registry is a custom Prometheus registry that avoids duplicate registration panics.
var Registry = newRegistry()

func newRegistry() *prometheus.Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	r.MustRegister(collectors.NewGoCollector())
	return r
}

// GRPCMetrics holds Prometheus metrics for gRPC services.
type GRPCMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

// NewGRPCMetrics registers and returns standard gRPC metrics.
func NewGRPCMetrics(namespace string) *GRPCMetrics {
	factory := promauto.With(Registry)
	return &GRPCMetrics{
		RequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "grpc",
			Name:      "requests_total",
			Help:      "Total number of gRPC requests",
		}, []string{"method", "code"}),

		RequestDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "grpc",
			Name:      "request_duration_seconds",
			Help:      "gRPC request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method"}),
	}
}
