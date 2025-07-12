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
	spanCtx, span := r.tracer.Start(ctx, "Get")
	defer span.End()

	cached, err := r.client.Get(spanCtx, key).Result()
	if err == redis.Nil {
		// Cache miss
		r.logger.WithContext(spanCtx).WithError(err).Info("Cache miss")
		return "", false
	}

	if err != nil {
		// Redis error
		r.logger.WithContext(spanCtx).WithError(err).Error("Redis gor error")
		return "", false
	}

	r.logger.WithContext(spanCtx).WithField("key", key).Info("Redis cache hit")
	return cached, true
}

// Set marshals value to JSON and stores it in Redis with TTL.
func (r *RedisService) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) (string, error) {
	spanCtx, span := r.tracer.Start(ctx, "Set")
	defer span.End()

	json, err := json.Marshal(data)
	if err != nil {
		r.logger.WithContext(spanCtx).WithError(err).Warn("Failed to marshal user response")
		return "", err
	}
	if err := r.client.Set(spanCtx, key, json, ttl).Err(); err != nil {
		r.logger.WithContext(spanCtx).WithError(err).Error("Failed to store data to redis")
		return "", err
	}

	return string(json), nil
}
