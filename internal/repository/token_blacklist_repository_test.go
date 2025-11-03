package repository

import (
    "context"
    "testing"
    "time"

    "go-starter-template/internal/constant"

    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/require"
)

// Table-driven tests for in-memory TokenBlacklist
func TestTokenBlacklist_InMemory(t *testing.T) {
    type tc struct {
        name   string
        token  string
        action func(tb *TokenBlacklist) (bool, error)
        assert func(t *testing.T, got bool, err error)
    }

    cases := []tc{
        {
            name:  "AddAndCheckTrue",
            token: "tok1",
            action: func(tb *TokenBlacklist) (bool, error) {
                require.NoError(t, tb.Add("tok1", 10*time.Second))
                return tb.IsBlacklisted("tok1")
            },
            assert: func(t *testing.T, got bool, err error) {
                require.NoError(t, err)
                require.True(t, got)
            },
        },
        {
            name:  "CheckFalseWhenNotAdded",
            token: "tok2",
            action: func(tb *TokenBlacklist) (bool, error) {
                return tb.IsBlacklisted("tok2")
            },
            assert: func(t *testing.T, got bool, err error) {
                require.NoError(t, err)
                require.False(t, got)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            tb := NewTokenBlacklist()
            got, err := c.action(tb)
            c.assert(t, got, err)
        })
    }
}

// Table-driven tests for RedisTokenBlacklist Add and IsBlacklisted
func TestRedisTokenBlacklist(t *testing.T) {
    type tc struct {
        name      string
        token     string
        tokenType constant.TokenType
        duration  time.Duration
        mutate    func(r *RedisTokenBlacklist)
        assert    func(t *testing.T, r *RedisTokenBlacklist)
    }

    // Start a MiniRedis server for integration-like tests
    mr, err := miniredis.Run()
    require.NoError(t, err)
    defer mr.Close()

    client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    repo := NewRedisTokenBlacklist(client)

    cases := []tc{
        {
            name:      "AddAccessAndCheckTrue",
            token:     "tokA",
            tokenType: constant.TokenTypeAccess,
            duration:  30 * time.Second,
            mutate: func(r *RedisTokenBlacklist) {
                require.NoError(t, r.Add("tokA", constant.TokenTypeAccess, 30*time.Second))
            },
            assert: func(t *testing.T, r *RedisTokenBlacklist) {
                ok, err := r.IsBlacklisted("tokA", constant.TokenTypeAccess)
                require.NoError(t, err)
                require.True(t, ok)
                // Also ensure key exists in Redis with expected pattern
                keys, _ := client.Keys(context.Background(), "blacklist:*:*").Result()
                // At least one key exists; we won't strictly match due to token values
                require.NotEmpty(t, keys)
            },
        },
        {
            name:      "AddRefreshAndCheckTrue",
            token:     "tokR",
            tokenType: constant.TokenTypeRefresh,
            duration:  10 * time.Second,
            mutate: func(r *RedisTokenBlacklist) {
                require.NoError(t, r.Add("tokR", constant.TokenTypeRefresh, 10*time.Second))
            },
            assert: func(t *testing.T, r *RedisTokenBlacklist) {
                ok, err := r.IsBlacklisted("tokR", constant.TokenTypeRefresh)
                require.NoError(t, err)
                require.True(t, ok)
            },
        },
        {
            name:      "NotBlacklisted",
            token:     "missing",
            tokenType: constant.TokenTypeAccess,
            duration:  5 * time.Second,
            mutate:    func(r *RedisTokenBlacklist) {},
            assert: func(t *testing.T, r *RedisTokenBlacklist) {
                ok, err := r.IsBlacklisted("missing", constant.TokenTypeAccess)
                require.NoError(t, err)
                require.False(t, ok)
            },
        },
        {
            name:      "TTLExpires",
            token:     "expire",
            tokenType: constant.TokenTypeAccess,
            duration:  1 * time.Second,
            mutate: func(r *RedisTokenBlacklist) {
                require.NoError(t, r.Add("expire", constant.TokenTypeAccess, 1*time.Second))
                // Fast-forward MiniRedis time to trigger TTL expiry
                mr.FastForward(2 * time.Second)
            },
            assert: func(t *testing.T, r *RedisTokenBlacklist) {
                ok, err := r.IsBlacklisted("expire", constant.TokenTypeAccess)
                require.NoError(t, err)
                require.False(t, ok)
            },
        },
        {
            name:      "GetErrorBranch",
            token:     "errtoken",
            tokenType: constant.TokenTypeAccess,
            duration:  5 * time.Second,
            mutate: func(r *RedisTokenBlacklist) {
                // Close the MiniRedis server to force client.Get to return an error (not redis.Nil)
                mr.Close()
            },
            assert: func(t *testing.T, r *RedisTokenBlacklist) {
                _, err := r.IsBlacklisted("errtoken", constant.TokenTypeAccess)
                require.Error(t, err)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            // reset Redis between cases to avoid leakage
            mr.FlushAll()
            c.mutate(repo)
            c.assert(t, repo)
        })
    }
}