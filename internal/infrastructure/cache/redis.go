package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"turtle/config"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

// Connect establishes Redis connection
func Connect(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("✅ Redis connected successfully")
	return nil
}

// Close closes Redis connection
func Close() error {
	if Client == nil {
		return nil
	}
	return Client.Close()
}

// HealthCheck performs Redis health check
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("Redis not initialized")
	}
	return Client.Ping(ctx).Err()
}

// Cache utility functions

// Set sets a key-value pair with expiration
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return Client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value by key
func Get(ctx context.Context, key string) (string, error) {
	return Client.Get(ctx, key).Result()
}

// Delete deletes a key
func Delete(ctx context.Context, keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

// Exists checks if key exists
func Exists(ctx context.Context, keys ...string) (int64, error) {
	return Client.Exists(ctx, keys...).Result()
}

// Expire sets expiration on a key
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return Client.Expire(ctx, key, expiration).Err()
}

// Distributed Lock functions

// AcquireLock attempts to acquire a distributed lock
func AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return Client.SetNX(ctx, "lock:"+key, "1", ttl).Result()
}

// ReleaseLock releases a distributed lock
func ReleaseLock(ctx context.Context, key string) error {
	return Client.Del(ctx, "lock:"+key).Err()
}

// WithLock executes a function while holding a distributed lock
func WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	lockKey := "lock:" + key

	// Try to acquire lock with retries
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	var acquired bool
	var err error

	for i := 0; i < maxRetries; i++ {
		acquired, err = Client.SetNX(ctx, lockKey, "1", ttl).Result()
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}

		if acquired {
			break
		}

		// Wait before retry
		time.Sleep(retryDelay * time.Duration(i+1))
	}

	if !acquired {
		return fmt.Errorf("failed to acquire lock after %d retries", maxRetries)
	}

	// Ensure lock is released even if function panics
	defer func() {
		if err := Client.Del(ctx, lockKey).Err(); err != nil {
			log.Printf("failed to release lock %s: %v", lockKey, err)
		}
	}()

	// Execute function
	return fn()
}

// Rate Limiting functions

// CheckRateLimit checks if action is within rate limit
// Returns: allowed (bool), remaining (int), error
func CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	rateLimitKey := "ratelimit:" + key

	// Increment counter
	count, err := Client.Incr(ctx, rateLimitKey).Result()
	if err != nil {
		return false, 0, err
	}

	// Set expiration on first increment
	if count == 1 {
		if err := Client.Expire(ctx, rateLimitKey, window).Err(); err != nil {
			return false, 0, err
		}
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	allowed := count <= int64(limit)
	return allowed, remaining, nil
}

// Idempotency functions

// StoreIdempotencyResult stores the result of an idempotent operation
func StoreIdempotencyResult(ctx context.Context, key string, result interface{}, expiration time.Duration) error {
	return Client.Set(ctx, "idempotency:"+key, result, expiration).Err()
}

// GetIdempotencyResult retrieves the result of an idempotent operation
func GetIdempotencyResult(ctx context.Context, key string) (string, error) {
	return Client.Get(ctx, "idempotency:"+key).Result()
}

// Session functions

// StoreSession stores user session data
func StoreSession(ctx context.Context, sessionID string, data interface{}, expiration time.Duration) error {
	return Client.Set(ctx, "session:"+sessionID, data, expiration).Err()
}

// GetSession retrieves session data
func GetSession(ctx context.Context, sessionID string) (string, error) {
	return Client.Get(ctx, "session:"+sessionID).Result()
}

// DeleteSession deletes a session
func DeleteSession(ctx context.Context, sessionID string) error {
	return Client.Del(ctx, "session:"+sessionID).Err()
}

// Token Blacklist functions

// BlacklistToken adds a token to the blacklist
func BlacklistToken(ctx context.Context, token string, expiration time.Duration) error {
	return Client.Set(ctx, "blacklist:"+token, "1", expiration).Err()
}

// IsTokenBlacklisted checks if token is blacklisted
func IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	exists, err := Client.Exists(ctx, "blacklist:"+token).Result()
	return exists > 0, err
}

// Cache Statistics

// GetCacheStats returns Redis statistics
type CacheStats struct {
	ConnectedClients int64
	UsedMemory       string
	TotalKeys        int64
	Hits             int64
	Misses           int64
}

func GetCacheStats(ctx context.Context) (*CacheStats, error) {
	_, err := Client.Info(ctx, "stats", "memory", "clients").Result()
	if err != nil {
		return nil, err
	}

	// Get key count
	keyCount, err := Client.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}

	// Parse info string to get stats
	// This is a simplified version - in production you'd parse the actual INFO output
	return &CacheStats{
		ConnectedClients: 0,     // Parse from info
		UsedMemory:       "0MB", // Parse from info
		TotalKeys:        keyCount,
		Hits:             0, // Parse from info
		Misses:           0, // Parse from info
	}, nil
}
