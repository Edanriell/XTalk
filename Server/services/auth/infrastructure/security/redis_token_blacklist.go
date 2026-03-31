package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// RedisTokenBlacklist implements interfaces.TokenBlacklist
type RedisTokenBlacklist struct {
	client *redis.Client
}

func NewRedisTokenBlacklist(client *redis.Client) interfaces.TokenBlacklist {
	return &RedisTokenBlacklist{client: client}
}

func (b *RedisTokenBlacklist) Add(ctx context.Context, token string, expiry time.Duration) error {
	opCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key := blacklistKey(token)
	return b.client.Set(opCtx, key, "1", expiry).Err()
}

func (b *RedisTokenBlacklist) IsBlacklisted(ctx context.Context, token string) bool {
	opCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key := blacklistKey(token)
	result, err := b.client.Get(opCtx, key).Result()
	if err == redis.Nil {
		return false
	}
	if err != nil {
		// Fail-closed: treat Redis errors as "blacklisted" to prevent
		// revoked tokens from being accepted when Redis is unavailable.
		return true
	}
	return result == "1"
}

func (b *RedisTokenBlacklist) Remove(ctx context.Context, token string) error {
	opCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key := blacklistKey(token)
	return b.client.Del(opCtx, key).Err()
}

// blacklistKey returns a fixed-size Redis key by hashing the token with SHA-256.
func blacklistKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return "blacklist:" + hex.EncodeToString(h[:])
}
