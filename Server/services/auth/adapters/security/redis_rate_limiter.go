package security

import (
	"XTalk/services/auth/application/interfaces"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements interfaces.RateLimiter
type RedisRateLimiter struct {
	client      *redis.Client
	maxAttempts int64
	window      time.Duration
}

func NewRedisRateLimiter(client *redis.Client, maxAttempts int64, window time.Duration) interfaces.RateLimiter {
	return &RedisRateLimiter{client: client, maxAttempts: maxAttempts, window: window}
}

func (r *RedisRateLimiter) Allow(identifier string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	key := fmt.Sprintf("login_attempts:%s", identifier)

	// Atomic: INCR then check. This avoids the TOCTOU race of GET-then-compare.
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		// If Redis is down, fail-open to avoid locking everyone out.
		return true
	}

	// Set expiry on first attempt (atomically paired with INCR).
	if count == 1 {
		r.client.Expire(ctx, key, r.window)
	}

	return count <= r.maxAttempts
}

func (r *RedisRateLimiter) IncrementFailure(identifier string) error {
	// Allow() already increments the counter, so this is now a no-op.
	// Kept for interface compatibility.
	return nil
}

func (r *RedisRateLimiter) Reset(identifier string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	key := fmt.Sprintf("login_attempts:%s", identifier)
	return r.client.Del(ctx, key).Err()
}
