package abstractions

// RateLimiter is an application port for rate limiting
type RateLimiter interface {
	Allow(identifier string) bool
	IncrementFailure(identifier string) error
	Reset(identifier string) error
}
