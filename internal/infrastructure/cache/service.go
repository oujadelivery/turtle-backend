package cache

import (
	"context"
	"time"
)

// CacheService defines the interface for cache operations
type CacheService interface {
	// Rate limiting
	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error)

	// Basic operations
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error

	// Locks
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error

	// Token blacklist
	BlacklistToken(ctx context.Context, token string, expiration time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

// RedisCache implements CacheService using Redis
type RedisCache struct {
	// Uses the global Client from redis.go
}

// NewRedisCache creates a new Redis cache service
func NewRedisCache() CacheService {
	return &RedisCache{}
}

// Implement CacheService interface methods by delegating to existing functions

func (r *RedisCache) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	return CheckRateLimit(ctx, key, limit, window)
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return Set(ctx, key, value, expiration)
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return Get(ctx, key)
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	return Delete(ctx, keys...)
}

func (r *RedisCache) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return AcquireLock(ctx, key, ttl)
}

func (r *RedisCache) ReleaseLock(ctx context.Context, key string) error {
	return ReleaseLock(ctx, key)
}

func (r *RedisCache) BlacklistToken(ctx context.Context, token string, expiration time.Duration) error {
	return BlacklistToken(ctx, token, expiration)
}

func (r *RedisCache) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return IsTokenBlacklisted(ctx, token)
}
