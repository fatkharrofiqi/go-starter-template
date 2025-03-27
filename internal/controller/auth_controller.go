package controller

import (
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/logutil"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type AuthController struct {
	AuthService *service.AuthService
	Logger      *logrus.Logger
	Validation  *validation.Validation
}

func NewAuthController(authService *service.AuthService, logger *logrus.Logger, validator *validation.Validation) *AuthController {
	return &AuthController{authService, logger, validator}
}

func (c *AuthController) Login(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Logger, ctx, "Login").Info("Processing login request")

	var req dto.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Logger.WithError(err).Error("Failed to parse login request")
		return fiber.ErrBadRequest
	}

	if err := c.Validation.Validate(req); err != nil {
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

	if err := c.Validation.Validate(req); err != nil {
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

	if err := c.Validation.Validate(req); err != nil {
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

func (c *AuthController) Logout(ctx *fiber.Ctx) error {
	logutil.AccessLog(c.Logger, ctx, "Logout").Info("Processing logout request")

	req := new(dto.LogoutRequest)
	if err := ctx.BodyParser(req); err != nil {
		c.Logger.Error("Failed to parse header")
		return fiber.ErrUnauthorized
	}

	if err := c.Validation.Validate(req); err != nil {
		c.Logger.Error("Payload required for logout")
		return err
	}

	err := c.AuthService.Logout(ctx.UserContext(), req.AccessToken, req.RefreshToken)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to logout")
		return err
	}

	return ctx.JSON(dto.WebResponse[string]{Data: "Logout successfully"})
}
