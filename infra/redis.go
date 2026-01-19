package infra

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func InitRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := Redis.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	log.Println("✅ Redis connected")
}


// MISSING: Comprehensive rate limiting
// type RateLimiter struct {
//     redis *redis.Client
// }

// func (rl *RateLimiter) CheckRateLimit(key string, limit int, window time.Duration) error {
//     count, _ := rl.redis.Incr(ctx, key).Result()
    
//     if count == 1 {
//         rl.redis.Expire(ctx, key, window)
//     }
    
//     if count > int64(limit) {
//         return errors.New("Exceeding the rate of call.")
//     }
    
//     return nil
// }

// Apply rate limits
// - Login attempts: 5 per 15 minutes per IP
// - OTP requests: 3 per hour per phone/email
// - API calls: 100 per minute per user
// - Order creation: 10 per hour per user
// - Password reset: 3 per day per account