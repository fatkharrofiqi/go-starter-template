package controller

import (
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/apperrors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type AuthController struct {
	AuthService *service.AuthService
	Logger      *logrus.Logger
	Validation  *validation.Validation
	Tracer      trace.Tracer
}

func NewAuthController(authService *service.AuthService, logger *logrus.Logger, validator *validation.Validation) *AuthController {
	return &AuthController{authService, logger, validator, otel.Tracer("AuthController")}
}

func (c *AuthController) Login(ctx *fiber.Ctx) error {
	userContext, span := c.Tracer.Start(ctx.UserContext(), "Login")
	defer span.End()

	var req dto.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Logger.WithError(err).Error("Failed to parse login request")
		return apperrors.ErrBadRequest
	}

	if err := c.Validation.Validate(req); err != nil {
		c.Logger.WithError(err).Warn("Validation failed for login request")
		return err
	}

	accessToken, refreshToken, err := c.AuthService.Login(userContext, req)
	if err != nil {
		c.Logger.WithError(err).Warn("Invalid login attempt")
		return err
	}

	c.setRefreshTokenCookie(ctx, refreshToken)

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{Data: &dto.TokenResponse{
		AccessToken: accessToken,
	}})
}

func (c *AuthController) Register(ctx *fiber.Ctx) error {
	userContext, span := c.Tracer.Start(ctx.UserContext(), "Register")
	defer span.End()

	var req dto.RegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		c.Logger.WithError(err).Error("Failed to parse registration request")
		return apperrors.ErrBadRequest
	}

	if err := c.Validation.Validate(req); err != nil {
		c.Logger.WithError(err).Warn("Validation failed for registration request")
		return err
	}

	user, err := c.AuthService.Register(userContext, req)
	if err != nil {
		c.Logger.WithError(err).Warn("User registration failed")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.UserResponse]{Data: user})
}

func (c *AuthController) RefreshToken(ctx *fiber.Ctx) error {
	userContext, span := c.Tracer.Start(ctx.UserContext(), "RefreshToken")
	defer span.End()

	refreshToken := ctx.Cookies("refresh_token")
	if refreshToken == "" {
		c.Logger.Warn("Refresh token not found in cookies")
		return apperrors.ErrUnauthorized
	}

	// Receive both access and refresh token
	accessToken, newRefreshToken, err := c.AuthService.RefreshToken(userContext, refreshToken)
	if err != nil {
		c.Logger.WithError(err).Warn("Invalid refresh token attempt")
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
	userContext, span := c.Tracer.Start(ctx.UserContext(), "Logout")
	defer span.End()

	authHeader := ctx.Get("Authorization")
	if authHeader == "" {
		c.Logger.Warn("Authorization header not found")
		return apperrors.ErrUnauthorized
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.Logger.Warn("Bearer not found in Authorization header")
		return apperrors.ErrUnauthorized
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	if accessToken == "" {
		c.Logger.Warn("Access token not found in Authorization header")
		return apperrors.ErrUnauthorized
	}

	refreshToken := ctx.Cookies("refresh_token")
	if refreshToken == "" {
		c.Logger.Warn("Refresh token not found in cookies")
		return apperrors.ErrUnauthorized
	}

	err := c.AuthService.Logout(userContext, accessToken, refreshToken)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to logout")
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
		Expires:  time.Now().Add(c.AuthService.JwtService.RefreshTokenExpiration),
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
