package redis

import (
    "bytes"
    "io"
    "testing"
    "time"

    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/require"

    "go-starter-template/internal/config/env"
)

// helper to build a config with provided address and pool settings
func testConfig(addr string) *env.Config {
    cfg := &env.Config{}
    cfg.Redis.Address = addr
    cfg.Redis.Password = ""
    cfg.Redis.DB = 1
    cfg.Redis.Pool.Size = 10
    cfg.Redis.Pool.MinIdle = 2
    cfg.Redis.Pool.MaxIdle = 5
    cfg.Redis.Pool.Lifetime = 120 // seconds
    cfg.Redis.Pool.IdleTimeout = 60 // seconds
    return cfg
}

func TestNewRedis_PingSuccess_ReturnsClientAndOptions(t *testing.T) {
    // start in‑memory redis
    mr, err := miniredis.Run()
    require.NoError(t, err)
    defer mr.Close()

    cfg := testConfig(mr.Addr())

    log := logrus.New()
    log.SetOutput(io.Discard)

    client := NewRedis(log, cfg)
    require.NotNil(t, client)

    // assert options mapped from config
    opts := client.Options()
    require.Equal(t, cfg.Redis.Address, opts.Addr)
    require.Equal(t, cfg.Redis.Password, opts.Password)
    require.Equal(t, cfg.Redis.DB, opts.DB)
    require.Equal(t, cfg.Redis.Pool.Size, opts.PoolSize)
    require.Equal(t, cfg.Redis.Pool.MinIdle, opts.MinIdleConns)
    require.Equal(t, cfg.Redis.Pool.MaxIdle, opts.MaxIdleConns)
    require.Equal(t, time.Duration(cfg.Redis.Pool.Lifetime)*time.Second, opts.ConnMaxLifetime)
    require.Equal(t, time.Duration(cfg.Redis.Pool.IdleTimeout)*time.Second, opts.ConnMaxIdleTime)

    // assert constant timeouts configured in NewRedis
    require.Equal(t, 5*time.Second, opts.DialTimeout)
    require.Equal(t, 3*time.Second, opts.ReadTimeout)
    require.Equal(t, 3*time.Second, opts.WriteTimeout)

    // basic connectivity check
    err = client.Set(t.Context(), "k", "v", 0).Err()
    require.NoError(t, err)
    v, err := client.Get(t.Context(), "k").Result()
    require.NoError(t, err)
    require.Equal(t, "v", v)
}

func TestNewRedis_PingFails_TriggersFatalExit(t *testing.T) {
    // use invalid port to force immediate dial error
    cfg := testConfig("127.0.0.1:bad")

    log := logrus.New()
    buf := &bytes.Buffer{}
    log.SetOutput(buf)

    exitCalled := false
    log.ExitFunc = func(code int) { exitCalled = true; panic("exit") }

    defer func() {
        r := recover()
        if r == nil {
            t.Fatalf("expected panic from ExitFunc, got none")
        }
        if !exitCalled {
            t.Fatalf("expected ExitFunc to be called")
        }
        // ensure fatal message is present
        require.Contains(t, buf.String(), "failed to connect to redis")
    }()

    _ = NewRedis(log, cfg)
}