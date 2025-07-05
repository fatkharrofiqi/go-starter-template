package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RedisService struct {
	Client *redis.Client
	Logger *logrus.Logger
}

func NewRedisService(client *redis.Client, logger *logrus.Logger) *RedisService {
	return &RedisService{
		Client: client,
		Logger: logger,
	}
}

// Get retrieves a string JSON value from Redis result.
func (r *RedisService) Get(ctx context.Context, key string) (string, bool) {
	cached, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Cache miss
		return "", false
	}

	if err != nil {
		// Redis error
		r.Logger.WithError(err).Error("redis get error")
		return "", false
	}

	r.Logger.WithField("key", key).Info("redis cache hit")
	return cached, true
}

// Set marshals value to JSON and stores it in Redis with TTL.
func (r *RedisService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	jsonData, err := json.Marshal(value)
	if err != nil {
		r.Logger.WithError(err).Error("failed to marshal data for redis")
		return
	}

	if setErr := r.Client.Set(ctx, key, jsonData, ttl).Err(); setErr != nil {
		r.Logger.WithError(setErr).Error("failed to store data to redis")
	}
}
