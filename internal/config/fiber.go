package config

import (
	"go-starter-template/internal/utils/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/spf13/viper"
)

func NewFiber(config *viper.Viper) *fiber.App {
	var app = fiber.New(fiber.Config{
		AppName:      config.GetString("app.name"),
		ErrorHandler: NewErrorHandler(),
		Prefork:      config.GetBool("web.prefork"),
	})

	// Add recover middleware to catch panics and prevent app crashes
	app.Use(recover.New())

	return app
}

func NewErrorHandler() fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		if ve, ok := err.(*validation.ValidationError); ok {
			return ctx.Status(fiber.ErrBadRequest.Code).JSON(ve)
		}

		// Default error handling
		return ctx.Status(code).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
}
