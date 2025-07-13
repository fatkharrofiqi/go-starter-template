package service

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type RedisService struct {
	client *redis.Client
	logger *logrus.Logger
	tracer trace.Tracer
}

func NewRedisService(client *redis.Client, logger *logrus.Logger) *RedisService {
	return &RedisService{client, logger, otel.Tracer("RedisService")}
}

// Get retrieves a string JSON value from Redis result.
func (r *RedisService) Get(ctx context.Context, key string) (string, bool) {
	spanCtx, span := r.tracer.Start(ctx, "RedisService.Get")
	defer span.End()

	logger := r.logger.WithContext(spanCtx)

	cached, err := r.client.Get(spanCtx, key).Result()
	if err == redis.Nil {
		// Cache miss
		logger.WithError(err).Info("Cache miss")
		return "", false
	}

	if err != nil {
		// Redis error
		logger.WithError(err).Error("Redis gor error")
		return "", false
	}

	logger.WithField("key", key).Info("Redis cache hit")
	return cached, true
}

// Set marshals value to JSON and stores it in Redis with TTL.
func (r *RedisService) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) (string, error) {
	spanCtx, span := r.tracer.Start(ctx, "RedisService.Set")
	defer span.End()

	logger := r.logger.WithContext(spanCtx)

	json, err := json.Marshal(data)
	if err != nil {
		logger.WithError(err).Warn("Failed to marshal user response")
		return "", err
	}
	if err := r.client.Set(spanCtx, key, json, ttl).Err(); err != nil {
		logger.WithError(err).Error("Failed to store data to redis")
		return "", err
	}

	return string(json), nil
}
