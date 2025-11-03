package service

import (
    "context"
    "errors"
    "io"
    "testing"
    "time"

    jwt "github.com/golang-jwt/jwt/v5"
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/require"

    "go-starter-template/internal/config/env"
    "go-starter-template/internal/constant"
    "go-starter-template/internal/utils/errcode"
)

// helpers (duplicated here for local test scope)
func blTestEnvConfig() *env.Config {
    cfg := &env.Config{}
    cfg.JWT.Secret = "access-secret"
    cfg.JWT.RefreshSecret = "refresh-secret"
    cfg.JWT.AccessTokenExpiration = 60
    cfg.JWT.RefreshTokenExpiration = 120
    return cfg
}

func blTestLogger() *logrus.Logger {
    l := logrus.New()
    l.SetOutput(io.Discard)
    return l
}

// fake blacklist repository implementing interface for BlacklistService tests
type blFakeRepo struct {
    isBlacklisted func(tokenHash string, tokenType constant.TokenType) (bool, error)
    add           func(tokenHash string, tokenType constant.TokenType, d time.Duration) error
    addCalled     bool
    lastTTL       time.Duration
}

func (f *blFakeRepo) Add(token string, tokenType constant.TokenType, d time.Duration) error {
    f.addCalled = true
    f.lastTTL = d
    if f.add != nil {
        return f.add(token, tokenType, d)
    }
    return nil
}

func (f *blFakeRepo) IsBlacklisted(token string, tokenType constant.TokenType) (bool, error) {
    if f.isBlacklisted != nil {
        return f.isBlacklisted(token, tokenType)
    }
    return false, nil
}

func TestBlacklistService_IsTokenBlacklisted(t *testing.T) {
    type testcase struct {
        name      string
        setupRepo func(*blFakeRepo)
        assert    func(*testing.T, error)
    }

    cfg := blTestEnvConfig()
    log := blTestLogger()
    jwtSvc := NewJwtService(log, cfg)

    cases := []testcase{
        {
            name: "RedisGetError",
            setupRepo: func(f *blFakeRepo) {
                f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, errors.New("redis get") }
            },
            assert: func(t *testing.T, err error) { require.ErrorIs(t, err, errcode.ErrRedisGet) },
        },
        {
            name: "AlreadyBlacklisted",
            setupRepo: func(f *blFakeRepo) {
                f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return true, nil }
            },
            assert: func(t *testing.T, err error) { require.ErrorIs(t, err, errcode.ErrUnauthorized) },
        },
        {
            name:   "NotBlacklisted",
            assert: func(t *testing.T, err error) { require.NoError(t, err) },
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            f := &blFakeRepo{}
            if tc.setupRepo != nil {
                tc.setupRepo(f)
            }
            blSvc := NewBlacklistService(log, jwtSvc, f)

            err := blSvc.IsTokenBlacklisted(context.Background(), "token", constant.TokenTypeRefresh)
            tc.assert(t, err)
        })
    }
}

func TestBlacklistService_Add(t *testing.T) {
    type testcase struct {
        name        string
        token       string
        tokenType   constant.TokenType
        setupRepo   func(*blFakeRepo)
        before      func()
        after       func()
        assert      func(*testing.T, *blFakeRepo, error)
    }

    cfg := blTestEnvConfig()
    log := blTestLogger()
    jwtSvc := NewJwtService(log, cfg)

    // valid access token (TTL > 0)
    validAccess, err := jwtSvc.GenerateAccessToken(context.Background(), "u1")
    require.NoError(t, err)

    // expired access token
    expiredClaims := Claims{
        UUID: "u1",
        Type: "access",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString([]byte(cfg.GetAccessSecret()))
    require.NoError(t, err)

    // access token expiring exactly now (TTL ~= 0)
    expiresNowClaims := Claims{
        UUID: "u1",
        Type: "access",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now()),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    expiresNowToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiresNowClaims).SignedString([]byte(cfg.GetAccessSecret()))
    require.NoError(t, err)

    cases := []testcase{
        {
            name:      "ParseError_FallbackAddSuccess",
            token:     "bad-token",
            tokenType: constant.TokenTypeAccess,
            setupRepo: func(f *blFakeRepo) {
                f.add = func(_ string, _ constant.TokenType, d time.Duration) error {
                    require.Equal(t, 24*time.Hour, d)
                    return nil
                }
            },
            assert: func(t *testing.T, f *blFakeRepo, err error) {
                require.NoError(t, err)
                require.True(t, f.addCalled)
                require.Equal(t, 24*time.Hour, f.lastTTL)
            },
        },
        {
            name:      "ParseError_FallbackAddFailure",
            token:     "bad-token",
            tokenType: constant.TokenTypeAccess,
            setupRepo: func(f *blFakeRepo) {
                f.add = func(_ string, _ constant.TokenType, d time.Duration) error { return errors.New("redis set") }
            },
            assert: func(t *testing.T, _ *blFakeRepo, err error) {
                require.ErrorIs(t, err, errcode.ErrRedisSet)
            },
        },
        {
            name:      "ExpiredToken_FallbackAddSuccess",
            token:     expiredToken,
            tokenType: constant.TokenTypeAccess,
            setupRepo: func(f *blFakeRepo) {
                f.add = func(_ string, _ constant.TokenType, d time.Duration) error {
                    require.Equal(t, 24*time.Hour, d)
                    return nil
                }
            },
            assert: func(t *testing.T, f *blFakeRepo, err error) {
                require.NoError(t, err)
                require.True(t, f.addCalled)
                require.Equal(t, 24*time.Hour, f.lastTTL)
            },
        },
        {
            name:      "Parsed_TTLZero_SkipBlacklist",
            token:     expiresNowToken,
            tokenType: constant.TokenTypeAccess,
            before: func() {
                // Force ParseTokenClaims to return claims with ExpiresAt == now
                jwtSvc.SetParseClaims(func(ctx context.Context, token string, tokenType constant.TokenType) (*Claims, error) {
                    return &Claims{UUID: "u1", Type: "access", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now()), IssuedAt: jwt.NewNumericDate(time.Now())}}, nil
                })
            },
            after: func() { jwtSvc.SetParseClaims(nil) },
            assert: func(t *testing.T, f *blFakeRepo, err error) {
                require.NoError(t, err)
                require.False(t, f.addCalled)
            },
        },
        {
            name:      "Parsed_TTLPositive_AddSuccess",
            token:     validAccess,
            tokenType: constant.TokenTypeAccess,
            setupRepo: func(f *blFakeRepo) {
                f.add = func(_ string, _ constant.TokenType, d time.Duration) error {
                    require.True(t, d > 0)
                    return nil
                }
            },
            assert: func(t *testing.T, f *blFakeRepo, err error) {
                require.NoError(t, err)
                require.True(t, f.addCalled)
                require.True(t, f.lastTTL > 0)
            },
        },
        {
            name:      "Parsed_TTLPositive_AddFailure",
            token:     validAccess,
            tokenType: constant.TokenTypeAccess,
            setupRepo: func(f *blFakeRepo) {
                f.add = func(_ string, _ constant.TokenType, d time.Duration) error { return errors.New("redis set") }
            },
            assert: func(t *testing.T, _ *blFakeRepo, err error) {
                require.ErrorIs(t, err, errcode.ErrRedisSet)
            },
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            f := &blFakeRepo{}
            if tc.setupRepo != nil { tc.setupRepo(f) }
            blSvc := NewBlacklistService(log, jwtSvc, f)
            if tc.before != nil { tc.before() }

            err := blSvc.Add(context.Background(), tc.token, tc.tokenType)
            tc.assert(t, f, err)
            if tc.after != nil { tc.after() }
        })
    }
}