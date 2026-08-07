package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ipEntry struct {
	tokens    float64
	lastCheck time.Time
}

// RateLimiter is a simple per-IP token-bucket rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
	rate    float64 // tokens per second
	burst   int     // max tokens
	done    chan struct{}
}

// NewRateLimiter creates a rate limiter allowing `rate` req/s with `burst` max burst per IP.
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*ipEntry),
		rate:    rate,
		burst:   burst,
		done:    make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Close stops the background cleanup goroutine.
func (rl *RateLimiter) Close() {
	close(rl.done)
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			stale := time.Now().Add(-5 * time.Minute)
			for k, v := range rl.entries {
				if v.lastCheck.Before(stale) {
					delete(rl.entries, k)
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	e, ok := rl.entries[ip]
	if !ok {
		rl.entries[ip] = &ipEntry{tokens: float64(rl.burst) - 1, lastCheck: now}
		return true
	}

	elapsed := now.Sub(e.lastCheck).Seconds()
	e.tokens += elapsed * rl.rate
	if e.tokens > float64(rl.burst) {
		e.tokens = float64(rl.burst)
	}
	e.lastCheck = now

	if e.tokens < 1 {
		return false
	}
	e.tokens--
	return true
}

// GinMiddleware returns a Gin middleware that rejects requests exceeding the rate limit.
func (rl *RateLimiter) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
