package controller

import (
	"go-starter-template/internal/dto"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type WelcomeController struct {
	tracer trace.Tracer
}

// NewWelcomeController creates a new instance of WelcomeController
func NewWelcomeController() *WelcomeController {
	return &WelcomeController{otel.Tracer("WelcomeController")}
}

func (r *WelcomeController) Hello(ctx *fiber.Ctx) error {
	_, span := r.tracer.Start(ctx.UserContext(), "Hello")
	defer span.End()

	return ctx.JSON(dto.WebResponse[interface{}]{
		Data: map[string]string{
			"Message": "Welcome to Go Starter API!",
		},
	})
}
