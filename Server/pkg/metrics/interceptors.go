package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor that records
// request count and duration metrics.
func UnaryServerInterceptor(m *GRPCMetrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()

		code := status.Code(err).String()
		m.RequestsTotal.WithLabelValues(info.FullMethod, code).Inc()
		m.RequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

		return resp, err
	}
}
