package tracing

import (
	"google.golang.org/grpc/stats"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func NewServerHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

func NewClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}

func DialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(NewClientHandler()),
	}
}
