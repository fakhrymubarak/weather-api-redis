package redis

import (
	"context"
	"sync"

	redisv9 "github.com/redis/go-redis/v9"
	"github.com/yourusername/weather-api-redis/internal/config"
)

var (
	client *redisv9.Client
	once   sync.Once
)

func GetClient() *redisv9.Client {
	once.Do(func() {
		client = redisv9.NewClient(&redisv9.Options{
			Addr: config.GetRedisAddr(),
		})
	})
	return client
}

func GetContext() context.Context {
	return context.Background()
}

// ResetClientForTest resets the Redis client singleton. Use only in tests.
func ResetClientForTest() {
	once = sync.Once{}
	client = nil
}
