package app

import (
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/validation"
	webcfg "go-starter-template/internal/config/web"
	"go-starter-template/internal/dto"
)

// Table-driven tests to verify Bootstrap wires routes and middleware correctly.
func TestApp_Bootstrap(t *testing.T) {
	// Set up shared dependencies
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mr := miniredis.RunT(t)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	cfg := &env.Config{}
	cfg.App.Name = "TestApp"
	cfg.Web.Prefork = false
	// Use specific origin to avoid wildcard+credentials panic in CORS
	cfg.Web.Cors.AllowOrigins = "http://example.com"
	// Minimal JWT config for service construction
	cfg.JWT.Secret = "access_secret"
	cfg.JWT.RefreshSecret = "refresh_secret"
	cfg.JWT.AccessTokenExpiration = 60
	cfg.JWT.RefreshTokenExpiration = 120
	cfg.JWT.CsrfTokenExpiration = 900

	// Use the project-provided Fiber constructor to get global error handler
	fib := webcfg.NewFiber(cfg)

	validator := validation.NewValidation()
	boot := NewApp(logger, cfg, db, fib, validator, rdb)
	boot.Bootstrap()

	type testcase struct {
		name         string
		method       string
		path         string
		setupReq     func(*http.Request)
		expectStatus int
		assert       func(*testing.T, *http.Response)
	}

	cases := []testcase{
		{
			name:         "WelcomeRoute_ReturnsJSON",
			method:       http.MethodGet,
			path:         "/",
			expectStatus: http.StatusOK,
			assert: func(t *testing.T, resp *http.Response) {
				require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
				var out dto.WebResponse[map[string]string]
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				require.Equal(t, "Welcome to Go Starter API!", out.Data["Message"])
			},
		},
		{
			name:         "AuthLogin_BadRequestOnEmptyBody",
			method:       http.MethodPost,
			path:         "/api/auth/login",
			expectStatus: http.StatusBadRequest,
			assert: func(t *testing.T, resp *http.Response) {
				var out dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				// Empty body triggers BodyParser error -> mapped to bad request
				require.Equal(t, "bad request", out.Message)
			},
		},
		{
			name:         "UsersList_UnauthorizedWithoutToken",
			method:       http.MethodGet,
			path:         "/api/users/",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "AuthLogout_UnauthorizedWithoutHeader",
			method:       http.MethodPost,
			path:         "/api/auth/logout",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "RefreshToken_UnauthorizedWithoutCookie",
			method:       http.MethodPost,
			path:         "/api/auth/refresh-token",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.setupReq != nil {
				tc.setupReq(req)
			}
			resp, err := fib.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, tc.expectStatus, resp.StatusCode)
			if tc.assert != nil {
				tc.assert(t, resp)
			}
		})
	}
}

// Verify Run starts the server and can be shut down gracefully.
func TestApp_Run_StartAndShutdown(t *testing.T) {
    db, _, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    mr := miniredis.RunT(t)
    defer mr.Close()
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

    logger := logrus.New()
    logger.SetOutput(io.Discard)

    cfg := &env.Config{}
    cfg.App.Name = "TestApp"
    cfg.Web.Prefork = false
    cfg.Web.Cors.AllowOrigins = "http://example.com"
    // Use ephemeral port to avoid conflicts
    cfg.Web.Port = 0
    cfg.JWT.Secret = "access_secret"
    cfg.JWT.RefreshSecret = "refresh_secret"
    cfg.JWT.AccessTokenExpiration = 60
    cfg.JWT.RefreshTokenExpiration = 120
    cfg.JWT.CsrfTokenExpiration = 900

    fib := webcfg.NewFiber(cfg)
    validator := validation.NewValidation()
    boot := NewApp(logger, cfg, db, fib, validator, rdb)

    done := make(chan struct{})
    go func() {
        boot.Run()
        close(done)
    }()

    // Allow some time for server to start
    time.Sleep(50 * time.Millisecond)

    // Request shutdown and ensure run returns
    require.NoError(t, boot.web.Shutdown())

    select {
    case <-done:
        // ok
    case <-time.After(2 * time.Second):
        t.Fatal("Run did not return after Shutdown")
    }
}
