package controller

import (
	"go-starter-template/internal/dto"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/logutil"
	"go-starter-template/internal/utils/validation"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type AuthController struct {
	AuthService *service.AuthService
	Log         *logrus.Logger
	Validator   *validator.Validate
}

func NewAuthController(authService *service.AuthService, logger *logrus.Logger, validator *validator.Validate) *AuthController {
	return &AuthController{authService, logger, validator}
}

func (c *AuthController) Login(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Log, ctx, "Login").Info("Processing login request")

	var req dto.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Log.WithError(err).Error("error parse request body")
		return err
	}

	if err := validation.ValidateStruct(c.Validator, req); err != nil {
		c.Log.WithError(err).Error("invalid request body")
		return err
	}

	token, err := c.AuthService.Login(ctx.UserContext(), req)
	if err != nil {
		c.Log.WithError(err).Error("invalid credentials")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{
		Data: token,
	})
}

func (c *AuthController) Register(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Log, ctx, "Register").Info("Processing registration request")

	var req dto.RegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Log.WithError(err).Error("error parse request body")
		return err
	}

	if err := validation.ValidateStruct(c.Validator, req); err != nil {
		c.Log.WithError(err).Error("invalid request body")
		return err
	}

	user, err := c.AuthService.Register(ctx.UserContext(), req)
	if err != nil {
		c.Log.WithError(err).Error("error user registration")
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

func (c *AuthController) RefreshToken(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Log, ctx, "RefreshToken").Info("Processing refresh token request")

	var req dto.RefreshTokenRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Log.WithError(err).Error("error parse request body")
		return err
	}

	if err := validation.ValidateStruct(c.Validator, req); err != nil {
		c.Log.WithError(err).Error("invalid request body")
		return err
	}

	token, err := c.AuthService.RefreshToken(ctx.UserContext(), req.RefreshToken)
	if err != nil {
		c.Log.WithError(err).Error("invalid credentials")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{
		Data: token,
	})
}
