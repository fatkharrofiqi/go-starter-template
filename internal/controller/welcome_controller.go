package controller

import (
	"go-starter-template/internal/dto"

	"github.com/gofiber/fiber/v2"
)

type WelcomeController struct{}

// NewWelcomeController creates a new instance of WelcomeController
func NewWelcomeController() *WelcomeController {
	return &WelcomeController{}
}

func (r *WelcomeController) Hello(ctx *fiber.Ctx) error {
	return ctx.JSON(dto.WebResponse[interface{}]{
		Data: map[string]string{
			"Message": "Welcome to Go Starter API!",
		},
	})
}
