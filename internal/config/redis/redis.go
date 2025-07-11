package redis

import (
	"context"
	"go-starter-template/internal/config/env"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// NewRedis initializes and returns a Redis client connection
func NewRedis(log *logrus.Logger, config *env.Config) *redis.Client {
	addr := config.Redis.Address
	password := config.Redis.Password
	db := config.Redis.DB

	poolSize := config.Redis.Pool.Size
	minIdleConns := config.Redis.Pool.MinIdle
	maxIdleConns := config.Redis.Pool.MaxIdle
	connMaxLifetime := config.Redis.Pool.Lifetime
	connMaxIdleTime := config.Redis.Pool.IdleTimeout

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

		// Connection pool settings
		PoolSize:        poolSize,
		MinIdleConns:    minIdleConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: time.Duration(connMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(connMaxIdleTime) * time.Second,

		// Connection timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.WithError(err).Fatal("failed to connect to redis")
	}

	log.Info("Redis connection established successfully")
	return rdb
}
