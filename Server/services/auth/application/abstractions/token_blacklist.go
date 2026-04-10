package abstractions

import (
	"context"
	"time"
)

// TokenBlacklist is an application port for token blacklisting
type TokenBlacklist interface {
	Add(ctx context.Context, token string, expiry time.Duration) error
	IsBlacklisted(ctx context.Context, token string) bool
	Remove(ctx context.Context, token string) error
}
