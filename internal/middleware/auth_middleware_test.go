package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"go-starter-template/internal/config/env"
	"go-starter-template/internal/constant"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errcode"
)

// testLogger returns a logger that discards output
func testLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// testEnvConfig constructs a minimal config with secrets and expirations
func testEnvConfig() *env.Config {
	cfg := &env.Config{}
	cfg.JWT.Secret = "access_secret"
	cfg.JWT.RefreshSecret = "refresh_secret"
	cfg.JWT.AccessTokenExpiration = 60
	cfg.JWT.RefreshTokenExpiration = 120
	cfg.JWT.CsrfTokenExpiration = 900
	return cfg
}

// fake blacklist repo implementing TokenBlacklistRepository
type fakeBLRepo struct {
	isBlacklisted func(tokenHash string, tokenType constant.TokenType) (bool, error)
	add           func(tokenHash string, tokenType constant.TokenType, d time.Duration) error // unused here
}

func (f *fakeBLRepo) Add(token string, tokenType constant.TokenType, d time.Duration) error {
	if f.add != nil {
		return f.add(token, tokenType, d)
	}
	return nil
}

func (f *fakeBLRepo) IsBlacklisted(token string, tokenType constant.TokenType) (bool, error) {
	if f.isBlacklisted != nil {
		return f.isBlacklisted(token, tokenType)
	}
	return false, nil
}

// TestAuthMiddleware covers error and success paths using table-driven tests
func TestAuthMiddleware(t *testing.T) {
	type testcase struct {
		name         string
		header       string
		setupBL      func(*fakeBLRepo)
		expectStatus int
		assert       func(*testing.T, *http.Response)
	}

	logger := testLogger()
	cfg := testEnvConfig()
	jwtSvc := service.NewJwtService(logger, cfg)
	f := &fakeBLRepo{}
	blSvc := service.NewBlacklistService(logger, jwtSvc, f)

	// Build Fiber app with error handler mapping errcodes
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
		if code, ok := errcode.GetHTTPStatus(err); ok {
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}})

	// Protected route applying middleware
	app.Get("/protected", AuthMiddleware(jwtSvc, blSvc, logger), func(c *fiber.Ctx) error {
		// On success, claims should be present
		claims := c.Locals("auth")
		if claims == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "claims missing")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	// Prepare a valid access token for success case
	validToken, err := jwtSvc.GenerateAccessToken(context.Background(), "u123")
	require.NoError(t, err)

	cases := []testcase{
		{
			name:         "MissingHeader",
			header:       "",
			expectStatus: fiber.StatusUnauthorized,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			},
		},
		{
			name:   "EmptyToken",
			header: "Bearer    ", // spaces trimmed -> empty token -> ErrAccessTokenMissing
			setupBL: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
			},
			expectStatus: fiber.StatusUnauthorized,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			},
		},
		{
			name:         "TooShortHeader",
			header:       "Bearer ", // len < minAuthLen
			expectStatus: fiber.StatusUnauthorized,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			},
		},
		{
			name:         "InvalidPrefix",
			header:       "Token abc",
			expectStatus: fiber.StatusUnauthorized,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			},
		},
		{
			name:   "Blacklisted",
			header: "Bearer some-token",
			setupBL: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return true, nil }
			},
			expectStatus: fiber.StatusUnauthorized,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			},
		},
		{
			name:   "BlacklistErrorRedis",
			header: "Bearer other-token",
			setupBL: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, errcode.ErrRedisGet }
			},
			expectStatus: fiber.StatusInternalServerError,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
			},
		},
		{
			name:   "InvalidTokenMapsToExpired",
			header: "Bearer invalid-token",
			setupBL: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
			},
			expectStatus: fiber.StatusUnauthorized,
			assert: func(t *testing.T, resp *http.Response) {
				// Middleware maps any validation error to ErrTokenIsExpired (401)
				require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			},
		},
		{
			name:   "Success",
			header: "Bearer " + validToken,
			setupBL: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
			},
			expectStatus: fiber.StatusOK,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, fiber.StatusOK, resp.StatusCode)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset f behavior per test
			f.isBlacklisted = nil
			if tc.setupBL != nil {
				tc.setupBL(f)
			}

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, tc.expectStatus, resp.StatusCode)
			if tc.assert != nil {
				tc.assert(t, resp)
			}
		})
	}
}

// TestGetUser verifies retrieving claims from Fiber locals
func TestGetUser(t *testing.T) {
	app := fiber.New()
	app.Get("/me", func(c *fiber.Ctx) error {
		c.Locals("auth", &service.Claims{UUID: "u42", Type: "access"})
		got := GetUser(c)
		require.NotNil(t, got)
		require.Equal(t, "u42", got.UUID)
		require.Equal(t, "access", got.Type)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}
