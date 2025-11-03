package service

import (
    "context"
    "crypto/rand"
    "crypto/rsa"
    "errors"
    "testing"
    "time"

    "go-starter-template/internal/config/env"
    "go-starter-template/internal/constant"
    "go-starter-template/internal/utils/errcode"

    "github.com/golang-jwt/jwt/v5"
    "github.com/stretchr/testify/require"
)

// Helper: make valid access token for tests
func makeAccessToken(t *testing.T, cfg *env.Config, uuid string, exp time.Time) string {
    t.Helper()
    claims := Claims{
        UUID: uuid,
        Type: "access",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(exp),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.GetAccessSecret()))
    require.NoError(t, err)
    return tok
}

// Helper: make valid refresh token for tests
func makeRefreshToken(t *testing.T, cfg *env.Config, uuid string, exp time.Time) string {
    t.Helper()
    claims := Claims{
        UUID: uuid,
        Type: "refresh",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(exp),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.GetRefreshSecret()))
    require.NoError(t, err)
    return tok
}

func TestJwtService_GenerateAccessToken(t *testing.T) {
    cfg := testEnvConfig()
    logger := testLogger()
    svc := NewJwtService(logger, cfg)

    type tc struct {
        name   string
        before func()
        after  func()
        assert func(t *testing.T, token string, err error)
    }

    cases := []tc{
        {
            name: "Success",
            assert: func(t *testing.T, token string, err error) {
                require.NoError(t, err)
                require.NotEmpty(t, token)
                // Validate claims
                claims, vErr := svc.ValidateAccessToken(context.Background(), token)
                require.NoError(t, vErr)
                require.Equal(t, "access", claims.Type)
                require.NotEmpty(t, claims.UUID)
            },
        },
        {
            name: "FailingSignMethod",
            before: func() { svc.SetAccessMethod(failingSignMethod{}) },
            after:  func() { svc.SetAccessMethod(jwt.SigningMethodHS256) },
            assert: func(t *testing.T, token string, err error) {
                require.Error(t, err)
                require.Empty(t, token)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            if c.before != nil { c.before() }
            token, err := svc.GenerateAccessToken(context.Background(), "u1")
            if c.after != nil { c.after() }
            c.assert(t, token, err)
        })
    }
}

func TestJwtService_GenerateRefreshToken(t *testing.T) {
    cfg := testEnvConfig()
    logger := testLogger()
    svc := NewJwtService(logger, cfg)

    type tc struct {
        name   string
        before func()
        after  func()
        assert func(t *testing.T, token string, err error)
    }

    cases := []tc{
        {
            name: "Success",
            assert: func(t *testing.T, token string, err error) {
                require.NoError(t, err)
                require.NotEmpty(t, token)
                claims, vErr := svc.ValidateRefreshToken(context.Background(), token)
                require.NoError(t, vErr)
                require.Equal(t, "refresh", claims.Type)
                require.NotEmpty(t, claims.UUID)
            },
        },
        {
            name: "FailingSignMethod",
            before: func() { svc.SetRefreshMethod(failingSignMethod{}) },
            after:  func() { svc.SetRefreshMethod(jwt.SigningMethodHS256) },
            assert: func(t *testing.T, token string, err error) {
                require.Error(t, err)
                require.Empty(t, token)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            if c.before != nil { c.before() }
            token, err := svc.GenerateRefreshToken(context.Background(), "u2")
            if c.after != nil { c.after() }
            c.assert(t, token, err)
        })
    }
}

func TestJwtService_ValidateAccessToken(t *testing.T) {
    cfg := testEnvConfig()
    logger := testLogger()
    svc := NewJwtService(logger, cfg)

    type tc struct {
        name   string
        token  string
        assert func(t *testing.T, claims *Claims, err error)
    }

    // Create tokens
    valid := makeAccessToken(t, cfg, "u3", time.Now().Add(1*time.Minute))
    expired := makeAccessToken(t, cfg, "u4", time.Now().Add(-1*time.Minute))

    // RS256 token to trigger unexpected sign method
    rsKey, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    rsClaims := Claims{UUID: "u5", Type: "access", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now())}}
    rsTok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, rsClaims).SignedString(rsKey)
    require.NoError(t, err)

    cases := []tc{
        {
            name:  "Valid",
            token: valid,
            assert: func(t *testing.T, claims *Claims, err error) {
                require.NoError(t, err)
                require.Equal(t, "u3", claims.UUID)
            },
        },
        {
            name:  "InvalidString",
            token: "not-a-token",
            assert: func(t *testing.T, _ *Claims, err error) {
                require.Error(t, err)
            },
        },
        {
            name:  "Expired",
            token: expired,
            assert: func(t *testing.T, _ *Claims, err error) {
                require.Error(t, err)
            },
        },
        {
            name:  "UnexpectedSignMethod",
            token: rsTok,
            assert: func(t *testing.T, _ *Claims, err error) {
                require.Error(t, err)
                require.True(t, errors.Is(err, errcode.ErrUnexpectedSignMethod))
            },
        },
        {
            name:  "TokenInvalidBranch",
            token: valid,
            assert: func(t *testing.T, _ *Claims, err error) {
                require.Error(t, err)
                require.True(t, errors.Is(err, errcode.ErrInvalidToken))
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            if c.name == "TokenInvalidBranch" {
                svc.SetValidateTokenOverride(func(tokenString string, claims *Claims, secretKey string) (*jwt.Token, error) {
                    // Return a token marked invalid without error
                    return &jwt.Token{Valid: false, Method: jwt.SigningMethodHS256}, nil
                })
                defer svc.SetValidateTokenOverride(nil)
            }
            claims, err := svc.ValidateAccessToken(context.Background(), c.token)
            c.assert(t, claims, err)
        })
    }
}

func TestJwtService_ValidateRefreshToken(t *testing.T) {
    cfg := testEnvConfig()
    logger := testLogger()
    svc := NewJwtService(logger, cfg)

    type tc struct {
        name   string
        token  string
        assert func(t *testing.T, claims *Claims, err error)
    }

    valid := makeRefreshToken(t, cfg, "u6", time.Now().Add(1*time.Minute))

    cases := []tc{
        {
            name:  "Valid",
            token: valid,
            assert: func(t *testing.T, claims *Claims, err error) {
                require.NoError(t, err)
                require.Equal(t, "u6", claims.UUID)
                require.Equal(t, "refresh", claims.Type)
            },
        },
        {
            name:  "InvalidString",
            token: "not-a-token",
            assert: func(t *testing.T, _ *Claims, err error) {
                require.Error(t, err)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            claims, err := svc.ValidateRefreshToken(context.Background(), c.token)
            c.assert(t, claims, err)
        })
    }
}

func TestJwtService_ParseTokenClaims(t *testing.T) {
    cfg := testEnvConfig()
    logger := testLogger()
    svc := NewJwtService(logger, cfg)

    access := makeAccessToken(t, cfg, "u7", time.Now().Add(1*time.Minute))
    refresh := makeRefreshToken(t, cfg, "u8", time.Now().Add(1*time.Minute))

    type tc struct {
        name      string
        token     string
        tokenType constant.TokenType
        before    func()
        after     func()
        assert    func(t *testing.T, claims *Claims, err error)
    }

    cases := []tc{
        {
            name:      "Access",
            token:     access,
            tokenType: constant.TokenTypeAccess,
            assert: func(t *testing.T, claims *Claims, err error) {
                require.NoError(t, err)
                require.Equal(t, "u7", claims.UUID)
            },
        },
        {
            name:      "Refresh",
            token:     refresh,
            tokenType: constant.TokenTypeRefresh,
            assert: func(t *testing.T, claims *Claims, err error) {
                require.NoError(t, err)
                require.Equal(t, "u8", claims.UUID)
            },
        },
        {
            name:      "UnsupportedType",
            token:     access,
            tokenType: constant.TokenType("other"),
            assert: func(t *testing.T, _ *Claims, err error) {
                require.Error(t, err)
            },
        },
        {
            name:      "OverrideUsed",
            token:     access,
            tokenType: constant.TokenTypeAccess,
            before: func() {
                svc.SetParseClaims(func(ctx context.Context, token string, tokenType constant.TokenType) (*Claims, error) {
                    return &Claims{UUID: "override", Type: "access", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(1*time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now())}}, nil
                })
            },
            after: func() { svc.SetParseClaims(nil) },
            assert: func(t *testing.T, claims *Claims, err error) {
                require.NoError(t, err)
                require.Equal(t, "override", claims.UUID)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            if c.before != nil { c.before() }
            claims, err := svc.ParseTokenClaims(context.Background(), c.token, c.tokenType)
            if c.after != nil { c.after() }
            c.assert(t, claims, err)
        })
    }
}

func TestJwtService_GenerateTokenHash(t *testing.T) {
    cfg := testEnvConfig()
    logger := testLogger()
    svc := NewJwtService(logger, cfg)

    type tc struct {
        name   string
        input  string
        expect string
    }

    cases := []tc{
        {name: "Empty", input: "", expect: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
        {name: "Hello", input: "hello", expect: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            got := svc.GenerateTokenHash(c.input)
            require.Equal(t, c.expect, got)
        })
    }
}