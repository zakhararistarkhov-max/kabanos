// Package redisx wraps the Redis client used for rate limiting and ephemeral
// caching. Keeping it behind a tiny package makes it trivial to swap the
// backing store or add tracing later.
package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kabanos/backend/internal/config"
)

// New constructs a Redis client and verifies connectivity.
func New(ctx context.Context, cfg config.Redis) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
