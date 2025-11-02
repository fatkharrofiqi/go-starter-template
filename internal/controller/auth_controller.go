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

const (
	refreshTokenCookieName = "refresh_token"
	bearerPrefix           = "Bearer "
)

var baseRefreshTokenCookie = fiber.Cookie{
	Name:     refreshTokenCookieName,
	HTTPOnly: true,
	Secure:   true,
	SameSite: "Strict",
	Path:     "/",
	Domain:   "",
}

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
	spanCtx, span := c.tracer.Start(ctx.UserContext(), "AuthController.Login")
	defer span.End()

	logger := c.logger.WithContext(spanCtx)

	_, parseSpan := c.tracer.Start(spanCtx, "ParseAndValidate")
	req := new(dto.LoginRequest)
	if err := c.validation.ParseAndValidate(ctx, req); err != nil {
		parseSpan.End()
		logger.WithError(err).Error("Failed to parse and validate login request")
		return err
	}
	parseSpan.End()

	accessToken, refreshToken, err := c.authService.Login(spanCtx, req)
	if err != nil {
		logger.WithError(err).Warn("Invalid login attempt")
		return err
	}

	_, setCookieSpan := c.tracer.Start(spanCtx, "SetCookie")
	c.setRefreshTokenCookie(ctx, refreshToken)
	setCookieSpan.End()

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{Data: &dto.TokenResponse{
		AccessToken: accessToken,
	}})
}

func (c *AuthController) Register(ctx *fiber.Ctx) error {
	spanCtx, span := c.tracer.Start(ctx.UserContext(), "AuthController.Register")
	defer span.End()

	logger := c.logger.WithContext(spanCtx)

	_, parseSpan := c.tracer.Start(spanCtx, "ParseAndValidate")
	req := new(dto.RegisterRequest)
	if err := c.validation.ParseAndValidate(ctx, req); err != nil {
		parseSpan.End()
		logger.WithError(err).Error("Failed to parse and validate register request")
		return err
	}
	parseSpan.End()

	user, err := c.authService.Register(spanCtx, req)
	if err != nil {
		logger.WithError(err).Warn("User registration failed")
		return err
	}

	return ctx.JSON(dto.WebResponse[*dto.UserResponse]{Data: user})
}

func (c *AuthController) RefreshToken(ctx *fiber.Ctx) error {
	spanCtx, span := c.tracer.Start(ctx.UserContext(), "AuthController.RefreshToken")
	defer span.End()

	logger := c.logger.WithContext(spanCtx)

	_, readCookeSpan := c.tracer.Start(spanCtx, "ReadCookie")
	refreshToken := ctx.Cookies(refreshTokenCookieName)
	if refreshToken == "" {
		readCookeSpan.End()
		logger.Warn("Missing refresh token cookie")
		return errcode.ErrUnauthorized
	}
	readCookeSpan.End()

	accessToken, newRefreshToken, err := c.authService.RefreshToken(spanCtx, refreshToken)
	if err != nil {
		logger.WithError(err).Warn("Invalid refresh token attempt")
		c.clearRefreshTokenCookie(ctx)
		return err
	}

	_, setCookieSpan := c.tracer.Start(spanCtx, "SetCookie")
	c.setRefreshTokenCookie(ctx, newRefreshToken)
	setCookieSpan.End()

	return ctx.JSON(dto.WebResponse[*dto.TokenResponse]{Data: &dto.TokenResponse{
		AccessToken: accessToken,
	}})
}

func (c *AuthController) Logout(ctx *fiber.Ctx) error {
    spanCtx, span := c.tracer.Start(ctx.UserContext(), "AuthController.Logout")
    defer span.End()

    logger := c.logger.WithContext(spanCtx)

    authHeader := strings.TrimSpace(ctx.Get("Authorization"))
    // Check for Bearer scheme ignoring trailing space
    if !strings.HasPrefix(authHeader, "Bearer") {
        logger.Warn("Invalid or missing Authorization header")
        return errcode.ErrUnauthorized
    }

    // Extract token after Bearer and trim any whitespace
    accessToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
    if accessToken == "" {
        logger.Warn("Access token is empty after Bearer prefix")
        return errcode.ErrUnauthorized
    }
	_, readCookeSpan := c.tracer.Start(spanCtx, "ReadCookie")
	refreshToken := ctx.Cookies(refreshTokenCookieName)
	if refreshToken == "" {
		readCookeSpan.End()
		logger.Warn("Missing refresh token cookie")
		return errcode.ErrUnauthorized
	}
	readCookeSpan.End()

	if err := c.authService.Logout(spanCtx, accessToken, refreshToken); err != nil {
		logger.WithError(err).Error("Logout failed")
		return err
	}

	_, clearCookieSpan := c.tracer.Start(spanCtx, "ClearCookie")
	c.clearRefreshTokenCookie(ctx)
	clearCookieSpan.End()

	return ctx.JSON(dto.WebResponse[string]{Data: "Logout successfully"})
}

// Helper to set refresh token cookie with secure options
func (c *AuthController) setRefreshTokenCookie(ctx *fiber.Ctx, refreshToken string) {
	cookie := baseRefreshTokenCookie
	cookie.Value = refreshToken
	cookie.Expires = time.Now().Add(c.config.GetRefreshTokenExpiration())

	ctx.Cookie(&cookie)
}

// Helper to clear refresh token cookie
func (c *AuthController) clearRefreshTokenCookie(ctx *fiber.Ctx) {
	cookie := baseRefreshTokenCookie
	cookie.Value = ""
	cookie.Expires = time.Now().Add(-1 * time.Hour)

	ctx.Cookie(&cookie)
}
