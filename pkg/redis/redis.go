package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,

		DialTimeout: 5 * time.Second,

		ReadTimeout: 3 * time.Second,

		WriteTimeout: 3 * time.Second,

		PoolTimeout: 4 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis connection failed: %w", err)
	}

	return client, nil
}
