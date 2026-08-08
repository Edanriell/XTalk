package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// idempotencyEntry stores the response for a previously seen idempotency key.
type idempotencyEntry struct {
	statusCode int
	body       []byte
	createdAt  time.Time
}

// IdempotencyStore is a simple in-memory store keyed by (userID, idempotency-key).
type IdempotencyStore struct {
	mu         sync.RWMutex
	entries    map[string]idempotencyEntry
	ttl        time.Duration
	maxEntries int
	done       chan struct{}
}

// NewIdempotencyStore creates the store and starts a background cleanup goroutine.
func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	s := &IdempotencyStore{
		entries:    make(map[string]idempotencyEntry),
		ttl:        ttl,
		maxEntries: 100_000,
		done:       make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Close stops the background cleanup goroutine.
func (s *IdempotencyStore) Close() {
	close(s.done)
}

func (s *IdempotencyStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, v := range s.entries {
				if now.Sub(v.createdAt) > s.ttl {
					delete(s.entries, k)
				}
			}
			s.mu.Unlock()
		case <-s.done:
			return
		}
	}
}

func (s *IdempotencyStore) cacheKey(userID, key string) string {
	h := sha256.Sum256([]byte(userID + ":" + key))
	return hex.EncodeToString(h[:])
}

// Idempotency returns a Gin middleware that deduplicates mutating requests
// using the Idempotency-Key header. Only applies to POST/PUT/PATCH.
func Idempotency(store *IdempotencyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to mutating methods.
		if c.Request.Method != http.MethodPost &&
			c.Request.Method != http.MethodPut &&
			c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		userID, _ := c.Get("userID")
		uid, _ := userID.(string)

		cacheKey := store.cacheKey(uid, key)

		// Check for cached response.
		store.mu.RLock()
		entry, found := store.entries[cacheKey]
		store.mu.RUnlock()

		if found {
			c.Data(entry.statusCode, "application/json", entry.body)
			c.Abort()
			return
		}

		// Record the response after the handler runs.
		writer := &responseRecorder{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = writer

		c.Next()

		store.mu.Lock()
		if len(store.entries) < store.maxEntries {
			store.entries[cacheKey] = idempotencyEntry{
				statusCode: writer.status,
				body:       writer.body,
				createdAt:  time.Now(),
			}
		}
		store.mu.Unlock()
	}
}

// responseRecorder captures the status code and body written by downstream handlers.
type responseRecorder struct {
	gin.ResponseWriter
	status int
	body   []byte
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}
