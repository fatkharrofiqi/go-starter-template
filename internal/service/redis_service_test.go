package service

import (
    "context"
    "errors"
    "io"
    "testing"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/require"
)

// fakeRedisClient satisfies redisClient for testing
type fakeRedisClient struct {
    getFunc func(ctx context.Context, key string) *redis.StringCmd
    setFunc func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

func (f *fakeRedisClient) Get(ctx context.Context, key string) *redis.StringCmd { return f.getFunc(ctx, key) }
func (f *fakeRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
    return f.setFunc(ctx, key, value, expiration)
}

func silentLogger() *logrus.Logger {
    l := logrus.New()
    l.SetLevel(logrus.ErrorLevel)
    l.SetOutput(io.Discard)
    return l
}

func TestRedisService_Get(t *testing.T) {
    logger := silentLogger()

    type tc struct {
        name   string
        setup  func() redisClient
        assert func(t *testing.T, val string, ok bool)
    }

    cases := []tc{
        {
            name: "CacheMiss",
            setup: func() redisClient {
                return &fakeRedisClient{
                    getFunc: func(ctx context.Context, key string) *redis.StringCmd {
                        cmd := redis.NewStringCmd(ctx)
                        cmd.SetErr(redis.Nil)
                        return cmd
                    },
                }
            },
            assert: func(t *testing.T, val string, ok bool) {
                require.False(t, ok)
                require.Equal(t, "", val)
            },
        },
        {
            name: "RedisError",
            setup: func() redisClient {
                return &fakeRedisClient{
                    getFunc: func(ctx context.Context, key string) *redis.StringCmd {
                        cmd := redis.NewStringCmd(ctx)
                        cmd.SetErr(errors.New("boom"))
                        return cmd
                    },
                }
            },
            assert: func(t *testing.T, val string, ok bool) {
                require.False(t, ok)
                require.Equal(t, "", val)
            },
        },
        {
            name: "Hit",
            setup: func() redisClient {
                return &fakeRedisClient{
                    getFunc: func(ctx context.Context, key string) *redis.StringCmd {
                        cmd := redis.NewStringCmd(ctx)
                        cmd.SetVal("{\"ok\":true}")
                        return cmd
                    },
                }
            },
            assert: func(t *testing.T, val string, ok bool) {
                require.True(t, ok)
                require.Equal(t, "{\"ok\":true}", val)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            svc := NewRedisService(c.setup(), logger)
            val, ok := svc.Get(context.Background(), "k")
            c.assert(t, val, ok)
        })
    }
}

func TestRedisService_Set(t *testing.T) {
    logger := silentLogger()

    type tc struct {
        name   string
        setup  func() redisClient
        input  any
        ttl    time.Duration
        assert func(t *testing.T, val string, err error)
    }

    cases := []tc{
        {
            name: "MarshalError",
            setup: func() redisClient { return &fakeRedisClient{} },
            input: func() {}, // functions are not json-marshallable
            ttl:   time.Second,
            assert: func(t *testing.T, val string, err error) {
                require.Error(t, err)
                require.Empty(t, val)
            },
        },
        {
            name: "RedisSetError",
            setup: func() redisClient {
                return &fakeRedisClient{
                    setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
                        cmd := redis.NewStatusCmd(ctx)
                        cmd.SetErr(errors.New("store failed"))
                        return cmd
                    },
                }
            },
            input: map[string]any{"ok": true},
            ttl:   time.Second,
            assert: func(t *testing.T, val string, err error) {
                require.Error(t, err)
                require.Empty(t, val)
            },
        },
        {
            name: "Success",
            setup: func() redisClient {
                return &fakeRedisClient{
                    setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
                        cmd := redis.NewStatusCmd(ctx)
                        cmd.SetVal("OK")
                        return cmd
                    },
                }
            },
            input: map[string]any{"ok": true},
            ttl:   2 * time.Second,
            assert: func(t *testing.T, val string, err error) {
                require.NoError(t, err)
                require.Equal(t, "{\"ok\":true}", val)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            svc := NewRedisService(c.setup(), logger)
            val, err := svc.Set(context.Background(), "k", c.input, c.ttl)
            c.assert(t, val, err)
        })
    }
}