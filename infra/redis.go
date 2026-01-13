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
