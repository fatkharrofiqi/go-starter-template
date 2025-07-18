package web

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/utils/errcode"

	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// NewFiber initializes a new Fiber app with custom configurations.
func NewFiber(config *env.Config) *fiber.App {
	var app = fiber.New(fiber.Config{
		AppName:      config.App.Name,
		ErrorHandler: newErrorHandler(),
		Prefork:      config.Web.Prefork,
		JSONEncoder:  json.Marshal,
		JSONDecoder:  json.Unmarshal,
	})

	// Recover middleware to prevent crashes from panics
	app.Use(recover.New())
	app.Use(middleware.Cors(config))
	app.Use(otelfiber.Middleware())

	return app
}

// newErrorHandler returns a structured global error handler.
func newErrorHandler() fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		response := dto.ErrorResponse{
			Message: "Internal server error",
		}

		// Check if the error exists in the custom error map
		if code, exists := errcode.GetHTTPStatus(err); exists {
			response.Message = err.Error()
			return ctx.Status(code).JSON(response)
		}

		// Handle go-playground validation errors
		if ve, ok := err.(*validation.ValidationError); ok {
			response.Message = "Validation failed"
			response.Errors = ve.Errors
			return ctx.Status(fiber.StatusBadRequest).JSON(response)
		}

		// Handle Fiber errors (e.g., JSON parsing)
		if e, ok := err.(*fiber.Error); ok {
			response.Message = e.Message
			return ctx.Status(e.Code).JSON(response)
		}

		return ctx.Status(fiber.StatusInternalServerError).JSON(response)
	}
}
