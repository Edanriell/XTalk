package circuitbreaker

import (
	"context"
	"sync"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Registry manages per-service circuit breakers so every gRPC connection
// gets protection without manual wiring in each handler.
type Registry struct {
	mu       sync.RWMutex
	breakers map[string]*GRPCCircuitBreaker
	log      *zap.Logger
	cfg      CBConfig
}

// CBConfig holds circuit breaker tuning parameters.
type CBConfig struct {
	FailureThreshold int
	WindowSize       int
	Delay            time.Duration
	SuccessThreshold int
}

// DefaultCBConfig returns sensible defaults.
func DefaultCBConfig() CBConfig {
	return CBConfig{
		FailureThreshold: 6,
		WindowSize:       10,
		Delay:            30 * time.Second,
		SuccessThreshold: 3,
	}
}

// NewRegistry creates a circuit breaker registry.
func NewRegistry(log *zap.Logger, opts ...CBConfig) *Registry {
	c := DefaultCBConfig()
	if len(opts) > 0 {
		c = opts[0]
	}
	return &Registry{
		breakers: make(map[string]*GRPCCircuitBreaker),
		log:      log.Named("circuitbreaker"),
		cfg:      c,
	}
}

// Get returns the circuit breaker for the given service, creating one if needed.
func (r *Registry) Get(serviceName string) *GRPCCircuitBreaker {
	r.mu.RLock()
	cb, ok := r.breakers[serviceName]
	r.mu.RUnlock()
	if ok {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok = r.breakers[serviceName]; ok {
		return cb
	}

	cb = newGRPCCircuitBreaker(serviceName, r.log, r.cfg)
	r.breakers[serviceName] = cb
	return cb
}

// DialOptions returns gRPC dial options with circuit breaking for the named service.
func (r *Registry) DialOptions(serviceName string) []grpc.DialOption {
	cb := r.Get(serviceName)
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(cb.UnaryClientInterceptor()),
	}
}

// GRPCCircuitBreaker wraps gRPC client calls with a failsafe-go circuit breaker.
type GRPCCircuitBreaker struct {
	breaker circuitbreaker.CircuitBreaker[any]
}

// isGRPCFailure returns true for gRPC error codes that indicate a service-level failure.
func isGRPCFailure(err error) bool {
	if err == nil {
		return false
	}
	code := status.Code(err)
	switch code {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted,
		codes.Aborted, codes.Internal, codes.Unknown:
		return true
	default:
		return false
	}
}

// newGRPCCircuitBreaker creates a circuit breaker for the given service.
func newGRPCCircuitBreaker(serviceName string, log *zap.Logger, cfg CBConfig) *GRPCCircuitBreaker {
	cb := circuitbreaker.NewBuilder[any]().
		HandleIf(func(_ any, err error) bool {
			return isGRPCFailure(err)
		}).
		WithFailureThresholdRatio(uint(cfg.FailureThreshold), uint(cfg.WindowSize)).
		WithDelay(cfg.Delay).
		WithSuccessThreshold(uint(cfg.SuccessThreshold)).
		OnStateChanged(func(e circuitbreaker.StateChangedEvent) {
			log.Warn("circuit breaker state changed",
				zap.String("service", serviceName),
				zap.String("from", e.OldState.String()),
				zap.String("to", e.NewState.String()),
			)
		}).
		Build()

	return &GRPCCircuitBreaker{breaker: cb}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor with circuit breaking.
// Uses a manual interceptor instead of failsafegrpc to avoid context cancellation issues.
func (gcb *GRPCCircuitBreaker) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if !gcb.breaker.TryAcquirePermit() {
			return status.Error(codes.Unavailable, "circuit breaker is open")
		}

		err := invoker(ctx, method, req, reply, cc, opts...)

		if isGRPCFailure(err) {
			gcb.breaker.RecordFailure()
		} else {
			gcb.breaker.RecordSuccess()
		}

		return err
	}
}

// IsOpen returns true if the circuit is currently open (service is unhealthy).
func (gcb *GRPCCircuitBreaker) IsOpen() bool {
	return gcb.breaker.IsOpen()
}
