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
	Client *redis.Client
	Logger *logrus.Logger
	Tracer trace.Tracer
}

func NewRedisService(client *redis.Client, logger *logrus.Logger) *RedisService {
	return &RedisService{
		Client: client,
		Logger: logger,
		Tracer: otel.Tracer("RedisService"),
	}
}

// Get retrieves a string JSON value from Redis result.
func (r *RedisService) Get(ctx context.Context, key string) (string, bool) {
	userContext, span := r.Tracer.Start(ctx, "Get")
	defer span.End()

	cached, err := r.Client.Get(userContext, key).Result()
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
func (r *RedisService) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) (string, error) {
	userContext, span := r.Tracer.Start(ctx, "Set")
	defer span.End()

	json, err := json.Marshal(data)
	if err != nil {
		r.Logger.WithError(err).Warn("failed to marshal user response")
		return "", err
	}
	if err := r.Client.Set(userContext, key, json, ttl).Err(); err != nil {
		r.Logger.WithError(err).Error("failed to store data to redis")
		return "", err
	}

	return string(json), nil
}
