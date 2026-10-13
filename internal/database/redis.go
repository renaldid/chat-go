package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/renaldid/chat-go/internal/config"
)

func NewRedisClient(
	ctx context.Context,
	cfg config.Config,
) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
