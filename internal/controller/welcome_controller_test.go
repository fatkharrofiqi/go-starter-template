package controller

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/require"

    "go-starter-template/internal/dto"
)

// TestWelcomeController_Hello verifies the welcome endpoint returns the expected JSON payload.
func TestWelcomeController_Hello(t *testing.T) {
    ctrl := NewWelcomeController()
    app := fiber.New()
    app.Get("/", ctrl.Hello)

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    resp, err := app.Test(req, -1)
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
    require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

    var out dto.WebResponse[map[string]string]
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
    require.Equal(t, "Welcome to Go Starter API!", out.Data["Message"])
}