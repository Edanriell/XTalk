package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TokenValidator is the interface the auth middleware needs.
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (userID string, email string, valid bool)
}

// tokenCacheEntry holds a cached validation result.
type tokenCacheEntry struct {
	userID    string
	email     string
	expiresAt time.Time
}

// tokenCache is a simple in-memory cache for validated tokens.
type tokenCache struct {
	mu         sync.RWMutex
	entries    map[string]tokenCacheEntry
	ttl        time.Duration
	maxEntries int
	done       chan struct{}
	log        *zap.Logger
}

func newTokenCache(ttl time.Duration, log *zap.Logger) *tokenCache {
	tc := &tokenCache{
		entries:    make(map[string]tokenCacheEntry),
		ttl:        ttl,
		maxEntries: 10_000, // cap to prevent unbounded growth
		done:       make(chan struct{}),
		log:        log,
	}
	// Background eviction of expired entries every minute.
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tc.evict()
			case <-tc.done:
				return
			}
		}
	}()
	return tc
}

// Close stops the background eviction goroutine.
func (tc *tokenCache) Close() {
	close(tc.done)
}

func (tc *tokenCache) get(token string) (string, string, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	entry, ok := tc.entries[token]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", "", false
	}
	return entry.userID, entry.email, true
}

func (tc *tokenCache) set(token, userID, email string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if len(tc.entries) >= tc.maxEntries {
		// Evict expired entries first; if still full, skip caching.
		now := time.Now()
		for k, v := range tc.entries {
			if now.After(v.expiresAt) {
				delete(tc.entries, k)
			}
		}
		if len(tc.entries) >= tc.maxEntries {
			tc.log.Warn("token cache full, unable to cache new entry", zap.Int("size", len(tc.entries)))
			return
		}
	}
	tc.entries[token] = tokenCacheEntry{
		userID:    userID,
		email:     email,
		expiresAt: time.Now().Add(tc.ttl),
	}
}

func (tc *tokenCache) evict() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	now := time.Now()
	for k, v := range tc.entries {
		if now.After(v.expiresAt) {
			delete(tc.entries, k)
		}
	}
}

// Auth returns a Gin middleware that validates Bearer tokens via the auth service,
// and a cleanup function that must be called on shutdown to stop background goroutines.
// Validated tokens are cached in-memory to avoid hitting the auth service on every request.
func Auth(validator TokenValidator, log *zap.Logger) (gin.HandlerFunc, func()) {
	cache := newTokenCache(2*time.Minute, log) // short TTL; keeps load off auth-service without masking revocations for long

	mw := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			return
		}

		token := parts[1]

		// Check cache first
		if userID, email, ok := cache.get(token); ok {
			c.Set("userID", userID)
			c.Set("email", email)
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		userID, email, valid := validator.ValidateToken(ctx, token)
		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		cache.set(token, userID, email)
		c.Set("userID", userID)
		c.Set("email", email)
		c.Next()
	}

	return mw, cache.Close
}
