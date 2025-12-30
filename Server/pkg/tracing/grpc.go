package tracing

import (
	"google.golang.org/grpc/stats"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// NewServerHandler returns a gRPC stats.Handler that creates spans for each
// incoming RPC using OpenTelemetry.
func NewServerHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

// NewClientHandler returns a gRPC stats.Handler that propagates trace context
// to downstream services using OpenTelemetry.
func NewClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}

// DialOptions returns gRPC dial options that add tracing to outgoing calls.
func DialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(NewClientHandler()),
	}
}
