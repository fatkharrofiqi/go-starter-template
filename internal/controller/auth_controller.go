package controller

import (
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errcode"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type AuthController struct {
	authService *service.AuthService
	logger      *logrus.Logger
	validation  *validation.Validation
	config      *env.Config
	tracer      trace.Tracer
}

func NewAuthController(authService *service.AuthService, logger *logrus.Logger, validator *validation.Validation, config *env.Config) *AuthController {
	return &AuthController{authService, logger, validator, config, otel.Tracer("AuthController")}
}

func (c *AuthController) Login(ctx *fiber.Ctx) error {
	userContext, span := c.tracer.Start(ctx.UserContext(), "Login")
	defer span.End()

	req := new(dto.LoginRequest)
	if err := c.validation.ParseAndValidate(ctx, req); err != nil {
		c.logger.WithError(err).Error("Failed to parse and validate login request")
		return err
	}

	accessToken, refreshToken, err := c.authService.Login(userContext, req)
	if err != nil {
		c.logger.WithError(err).Warn("Invalid login attempt")
		return err
	}

	c.setRefreshTokenCookie(ctx, refreshToken)

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{Data: &dto.TokenResponse{
		AccessToken: accessToken,
	}})
}

func (c *AuthController) Register(ctx *fiber.Ctx) error {
	userContext, span := c.tracer.Start(ctx.UserContext(), "Register")
	defer span.End()

	req := new(dto.RegisterRequest)
	if err := c.validation.ParseAndValidate(ctx, req); err != nil {
		c.logger.WithError(err).Error("Failed to parse and validate register request")
		return err
	}

	user, err := c.authService.Register(userContext, req)
	if err != nil {
		c.logger.WithError(err).Warn("User registration failed")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.UserResponse]{Data: user})
}

func (c *AuthController) RefreshToken(ctx *fiber.Ctx) error {
	userContext, span := c.tracer.Start(ctx.UserContext(), "RefreshToken")
	defer span.End()

	refreshToken := ctx.Cookies("refresh_token")
	if refreshToken == "" {
		c.logger.Warn("Refresh token not found in cookies")
		return errcode.ErrUnauthorized
	}

	// Receive both access and refresh token
	accessToken, newRefreshToken, err := c.authService.RefreshToken(userContext, refreshToken)
	if err != nil {
		c.logger.WithError(err).Warn("Invalid refresh token attempt")
		c.clearRefreshTokenCookie(ctx)
		return err
	}

	// Set new refresh token in cookie
	c.setRefreshTokenCookie(ctx, newRefreshToken)

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{Data: &dto.TokenResponse{
		AccessToken: accessToken,
	}})
}

func (c *AuthController) Logout(ctx *fiber.Ctx) error {
	userContext, span := c.tracer.Start(ctx.UserContext(), "Logout")
	defer span.End()

	authHeader := ctx.Get("Authorization")
	if authHeader == "" {
		c.logger.Warn("Authorization header not found")
		return errcode.ErrUnauthorized
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.logger.Warn("Bearer not found in Authorization header")
		return errcode.ErrUnauthorized
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	if accessToken == "" {
		c.logger.Warn("Access token not found in Authorization header")
		return errcode.ErrUnauthorized
	}

	refreshToken := ctx.Cookies("refresh_token")
	if refreshToken == "" {
		c.logger.Warn("Refresh token not found in cookies")
		return errcode.ErrUnauthorized
	}

	err := c.authService.Logout(userContext, accessToken, refreshToken)
	if err != nil {
		c.logger.WithError(err).Error("Failed to logout")
		return err
	}

	c.clearRefreshTokenCookie(ctx)

	return ctx.JSON(dto.WebResponse[string]{Data: "Logout successfully"})
}

// Helper method to set refresh token cookie with secure options
func (c *AuthController) setRefreshTokenCookie(ctx *fiber.Ctx, refreshToken string) {
	ctx.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(c.config.GetRefreshTokenExpiration()),
		HTTPOnly: true,     // Prevent XSS attacks
		Secure:   true,     // Only send over HTTPS
		SameSite: "Strict", // Prevent CSRF attacks
		Path:     "/",      // Available for all paths
		Domain:   "",       // Use current domain
	})
}

// Helper method to clear refresh token cookie
func (c *AuthController) clearRefreshTokenCookie(ctx *fiber.Ctx) {
	ctx.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // Set to past time
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
		Domain:   "",
	})
}
