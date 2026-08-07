package ratelimit

import (
	"context"
	"time"

	"github.com/failsafe-go/failsafe-go/ratelimiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewRateLimiter creates a bursty rate limiter that permits the given number
// of executions per interval using failsafe-go.
func NewRateLimiter(maxExecutions uint, interval time.Duration) ratelimiter.RateLimiter[any] {
	return ratelimiter.NewBurstyBuilder[any](maxExecutions, interval).Build()
}

// UnaryRateLimitInterceptor creates a gRPC unary server interceptor that
// rejects requests when the rate limit is exceeded.
func UnaryRateLimitInterceptor(rl ratelimiter.RateLimiter[any]) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !rl.TryAcquirePermit() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
