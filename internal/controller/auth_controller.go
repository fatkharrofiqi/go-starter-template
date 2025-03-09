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
	Logger      *logrus.Logger
	Validator   *validator.Validate
}

func NewAuthController(authService *service.AuthService, logger *logrus.Logger, validator *validator.Validate) *AuthController {
	return &AuthController{authService, logger, validator}
}

func (c *AuthController) Login(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Logger, ctx, "Login").Info("Processing login request")

	var req dto.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Logger.WithError(err).Error("Failed to parse login request")
		return fiber.ErrBadRequest
	}

	if err := validation.ValidateStruct(c.Validator, req); err != nil {
		c.Logger.WithError(err).Warn("Validation failed for login request")
		return err
	}

	token, err := c.AuthService.Login(ctx.UserContext(), req)
	if err != nil {
		c.Logger.WithError(err).Warn("Invalid login attempt")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{Data: token})
}

func (c *AuthController) Register(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Logger, ctx, "Register").Info("Processing registration request")

	var req dto.RegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Logger.WithError(err).Error("Failed to parse registration request")
		return fiber.ErrBadRequest
	}

	if err := validation.ValidateStruct(c.Validator, req); err != nil {
		c.Logger.WithError(err).Warn("Validation failed for registration request")
		return err
	}

	user, err := c.AuthService.Register(ctx.UserContext(), req)
	if err != nil {
		c.Logger.WithError(err).Warn("User registration failed")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.UserResponse]{Data: user})
}

func (c *AuthController) RefreshToken(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Logger, ctx, "RefreshToken").Info("Processing refresh token request")

	var req dto.RefreshTokenRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Logger.WithError(err).Error("Failed to parse refresh token request")
		return fiber.ErrBadRequest
	}

	if err := validation.ValidateStruct(c.Validator, req); err != nil {
		c.Logger.WithError(err).Warn("Validation failed for refresh token request")
		return err
	}

	token, err := c.AuthService.RefreshToken(ctx.UserContext(), req.RefreshToken)
	if err != nil {
		c.Logger.WithError(err).Warn("Invalid refresh token attempt")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{Data: token})
}
