package config

import (
	"go-starter-template/internal/utils/apperrors"
	"go-starter-template/internal/utils/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// NewFiber initializes a new Fiber app with custom configurations.
func NewFiber(config *viper.Viper, logger *logrus.Logger) *fiber.App {
	var app = fiber.New(fiber.Config{
		AppName:      config.GetString("app.name"),
		ErrorHandler: NewErrorHandler(logger),
		Prefork:      config.GetBool("web.prefork"),
	})

	// Recover middleware to prevent crashes from panics
	app.Use(recover.New())

	return app
}

// NewErrorHandler returns a structured global error handler.
func NewErrorHandler(logger *logrus.Logger) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		statusCode := fiber.StatusInternalServerError
		response := fiber.Map{"message": "Internal Server Error"}

		// Handle Fiber-specific errors first (e.g., fiber.ErrBadRequest, fiber.ErrUnauthorized)
		if e, ok := err.(*fiber.Error); ok {
			statusCode = e.Code
			response["message"] = e.Message
		}

		// Handle validation errors
		if ve, ok := err.(*validation.ValidationError); ok {
			statusCode = fiber.StatusBadRequest
			response = fiber.Map{
				"message": "Validation failed",
				"errors":  ve.Errors,
			}
		}

		// Check if the error exists in the custom error map
		if customStatus, exists := apperrors.GetHTTPStatus(err); exists {
			statusCode = customStatus
			response["message"] = err.Error()
		}

		// Log internal server errors for debugging
		if statusCode == fiber.StatusInternalServerError {
			logger.WithError(err).Error("Unexpected internal server error")
		} else {
			logger.WithError(err).Warn("Handled request error")
		}

		// Return JSON response with correct status code
		return ctx.Status(statusCode).JSON(response)
	}
}
