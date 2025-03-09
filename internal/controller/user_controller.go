package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/middleware"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/logutil"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	Log         *logrus.Logger
	UserService *service.UserService
}

func NewUserController(logger *logrus.Logger, userService *service.UserService) *UserController {
	return &UserController{logger, userService}
}

func (c *UserController) Me(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Log, ctx, "Me").Info("Processing me request")

	auth := middleware.GetUser(ctx)

	user, err := c.UserService.GetUser(ctx.UserContext(), auth.UUID)
	if err != nil {
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.UserResponse]{
		Data: &dto.UserResponse{
			UUID:      user.UUID,
			Name:      user.Name,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	})
}
