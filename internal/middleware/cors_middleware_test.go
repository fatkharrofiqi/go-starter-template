package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/require"

    "go-starter-template/internal/config/env"
)

// helper to build a minimal config
func corsTestConfig(allowOrigins string) *env.Config {
    cfg := &env.Config{}
    cfg.Web.Cors.AllowOrigins = allowOrigins
    return cfg
}

// Table-driven tests for CORS middleware covering preflight, actual, and disallowed origins
func TestCors_Table(t *testing.T) {
    const allowedOrigin = "https://allowed.example"
    cfg := corsTestConfig(allowedOrigin)

    // build app
    app := fiber.New()
    app.Use(Cors(cfg))
    app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
    app.Get("/data", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
    app.Get("/blocked", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

    type tc struct {
        name          string
        method        string
        path          string
        origin        string
        reqHeaders    map[string]string
        expectStatus  int
        expectHeaders map[string]string
    }

    cases := []tc{
        {
            name:   "PreflightAllowed",
            method: http.MethodOptions,
            path:   "/",
            origin: allowedOrigin,
            reqHeaders: map[string]string{
                "Access-Control-Request-Method":   "GET",
                "Access-Control-Request-Headers":  "Authorization",
            },
            expectStatus: fiber.StatusNoContent,
            expectHeaders: map[string]string{
                "Access-Control-Allow-Origin":      allowedOrigin,
                "Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
                "Access-Control-Allow-Headers":     "Origin,Content-Type,Accept,Authorization",
                "Access-Control-Allow-Credentials": "true",
            },
        },
        {
            name:   "ActualAllowed",
            method: http.MethodGet,
            path:   "/data",
            origin: allowedOrigin,
            expectStatus: fiber.StatusOK,
            expectHeaders: map[string]string{
                "Access-Control-Allow-Origin":      allowedOrigin,
                "Access-Control-Expose-Headers":    "Content-Length",
                "Access-Control-Allow-Credentials": "true",
            },
        },
        {
            name:   "DisallowedOriginActual",
            method: http.MethodGet,
            path:   "/blocked",
            origin: "https://other.example",
            expectStatus: fiber.StatusOK,
            expectHeaders: map[string]string{
                // header should be absent/empty for disallowed origin
                "Access-Control-Allow-Origin": "",
            },
        },
        {
            name:   "PreflightDisallowed",
            method: http.MethodOptions,
            path:   "/",
            origin: "https://other.example",
            reqHeaders: map[string]string{
                "Access-Control-Request-Method":  "GET",
                "Access-Control-Request-Headers": "Authorization",
            },
            expectStatus: fiber.StatusNoContent,
            expectHeaders: map[string]string{
                // origin not allowed
                "Access-Control-Allow-Origin": "",
            },
        },
    }

    for _, cse := range cases {
        t.Run(cse.name, func(t *testing.T) {
            req := httptest.NewRequest(cse.method, cse.path, nil)
            if cse.origin != "" {
                req.Header.Set("Origin", cse.origin)
            }
            for k, v := range cse.reqHeaders {
                req.Header.Set(k, v)
            }

            resp, err := app.Test(req, -1)
            require.NoError(t, err)
            require.Equal(t, cse.expectStatus, resp.StatusCode)

            for hk, hv := range cse.expectHeaders {
                require.Equal(t, hv, resp.Header.Get(hk), "header %s mismatch", hk)
            }
        })
    }
}