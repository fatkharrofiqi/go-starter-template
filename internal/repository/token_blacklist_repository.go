package repository

import (
	"context"
	"fmt"
	"go-starter-template/internal/constant"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklistRepository interface {
	Add(token string, tokenType constant.TokenType, duration time.Duration) error
	IsBlacklisted(token string, tokenType constant.TokenType) (bool, error)
}

type TokenBlacklist struct {
	// This could be a map, a database table, or any other storage mechanism.
	blacklist map[string]struct{}
	mutex     sync.RWMutex
}

func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{blacklist: make(map[string]struct{})}
}

func (tb *TokenBlacklist) Add(token string, duration time.Duration) error {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.blacklist[token] = struct{}{}
	return nil
}

func (tb *TokenBlacklist) IsBlacklisted(token string) (bool, error) {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	_, exists := tb.blacklist[token]
	return exists, nil
}

type RedisTokenBlacklist struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisTokenBlacklist(client *redis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{client, context.Background()}
}

func (r *RedisTokenBlacklist) Add(token string, tokenType constant.TokenType, duration time.Duration) error {
	return r.client.Set(r.ctx, fmt.Sprintf("blacklist:%s:%s", tokenType, token), "1", duration).Err()
}

func (r *RedisTokenBlacklist) IsBlacklisted(token string, tokenType constant.TokenType) (bool, error) {
	result, err := r.client.Get(r.ctx, fmt.Sprintf("blacklist:%s:%s", tokenType, token)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return result == "1", nil
}
