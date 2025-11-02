package web

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "fmt"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/require"

    "go-starter-template/internal/config/env"
    "go-starter-template/internal/config/validation"
    "go-starter-template/internal/dto"
    "go-starter-template/internal/utils/errcode"
)

// helper to build a new app with minimal config
func newTestApp() *fiber.App {
    cfg := &env.Config{}
    cfg.App.Name = "TestApp"
    cfg.Web.Prefork = false
    // Use a specific origin to avoid CORS middleware panic for insecure wildcard + credentials
    cfg.Web.Cors.AllowOrigins = "http://example.com"
    return NewFiber(cfg)
}

// Table-driven tests for the global error handler behavior
func TestNewFiber_ErrorHandler(t *testing.T) {
    type testcase struct {
        name         string
        handler      fiber.Handler
        expectStatus int
        assert       func(t *testing.T, resp *http.Response)
    }

    cases := []testcase{
        {
            name: "ErrcodeMapping_Unauthorized",
            handler: func(c *fiber.Ctx) error {
                return errcode.ErrUnauthorized
            },
            expectStatus: http.StatusUnauthorized,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.ErrorResponse
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "unauthorized", out.Message)
                require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
            },
        },
        {
            name: "ValidationError_MapsTo400",
            handler: func(c *fiber.Ctx) error {
                return &validation.ValidationError{Message: "ignored", Errors: map[string][]string{"name": {"name is required"}}}
            },
            expectStatus: http.StatusBadRequest,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.ErrorResponse
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "Validation failed", out.Message)
                require.Contains(t, out.Errors, "name")
                require.Contains(t, out.Errors["name"], "name is required")
            },
        },
        {
            name: "FiberError_UsesMessageAndStatus",
            handler: func(c *fiber.Ctx) error {
                return fiber.NewError(fiber.StatusBadRequest, "invalid body")
            },
            expectStatus: http.StatusBadRequest,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.ErrorResponse
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "invalid body", out.Message)
            },
        },
        {
            name: "FiberErr_InternalServer",
            handler: func(c *fiber.Ctx) error { return fiber.ErrInternalServerError },
            expectStatus: http.StatusInternalServerError,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.ErrorResponse
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                // Fiber error branch uses e.Message, which is capitalized by Fiber
                require.Equal(t, "Internal Server Error", out.Message)
            },
        },
        {
            name: "DefaultFallback_InternalServer",
            handler: func(c *fiber.Ctx) error { return fmt.Errorf("unexpected") },
            expectStatus: http.StatusInternalServerError,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.ErrorResponse
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                // Fallback branch keeps default message
                require.Equal(t, "Internal server error", out.Message)
            },
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            app := newTestApp()
            app.Get("/", tc.handler)

            req := httptest.NewRequest(http.MethodGet, "/", nil)
            resp, err := app.Test(req, -1)
            require.NoError(t, err)
            require.Equal(t, tc.expectStatus, resp.StatusCode)
            if tc.assert != nil {
                tc.assert(t, resp)
            }
        })
    }
}

// CORS preflight should set expected headers from middleware configuration
func TestNewFiber_CORS_Preflight(t *testing.T) {
    app := newTestApp()
    // any route (not used for OPTIONS, middleware handles it)
    app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

    req := httptest.NewRequest(http.MethodOptions, "/", nil)
    req.Header.Set("Origin", "http://example.com")
    req.Header.Set("Access-Control-Request-Method", "POST")
    req.Header.Set("Access-Control-Request-Headers", "Authorization")

    resp, err := app.Test(req, -1)
    require.NoError(t, err)
    // fiber/cors typically returns 204 for preflight
    require.Equal(t, http.StatusNoContent, resp.StatusCode)
    require.Equal(t, "http://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
    require.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
    require.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Authorization")
    require.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
}

// Recover middleware should prevent panics and delegate to global error handler
func TestNewFiber_RecoverMiddleware(t *testing.T) {
    app := newTestApp()
    app.Get("/panic", func(c *fiber.Ctx) error {
        panic("boom")
    })

    req := httptest.NewRequest(http.MethodGet, "/panic", nil)
    resp, err := app.Test(req, -1)
    require.NoError(t, err)
    require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

    var out dto.ErrorResponse
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
    require.Equal(t, "Internal server error", out.Message)
}