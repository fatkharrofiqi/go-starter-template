package middleware

import (
	"go-starter-template/internal/constant"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errcode"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const (
	bearerPrefix = "Bearer "
	bearerLen    = len(bearerPrefix)
	minAuthLen   = bearerLen + 1 // "Bearer " + at least 1 char
	authKey      = "auth"
)

func AuthMiddleware(jwtService *service.JwtService, blacklistService *service.BlacklistService, log *logrus.Logger) fiber.Handler {
	tracer := otel.Tracer("AuthMiddleware")
	return func(c *fiber.Ctx) error {
		spanCtx, span := tracer.Start(c.UserContext(), "AuthMiddleware")
		defer span.End()

		logger := log.WithContext(spanCtx)

		// Fast path for missing header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			logger.Error("authorization header missing")
			return errcode.ErrAuthorizationHeader
		}

		// Optimized prefix check with single length check
		if len(authHeader) < minAuthLen {
			logger.Warn("invalid authorization header format")
			return errcode.ErrBearerHeader
		}

		// Use unsafe string comparison for better performance
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			logger.Warn("invalid authorization header format")
			return errcode.ErrBearerHeader
		}

		// Extract token using slice operation
		accessToken := authHeader[bearerLen:]
		if accessToken == "" {
			logger.Warn("access token missing in header")
			return errcode.ErrAccessTokenMissing
		}

		// Check blacklist first
		if err := blacklistService.IsTokenBlacklisted(spanCtx, accessToken, constant.TokenTypeAccess); err != nil {
			logger.Warn("access token is blacklisted")
			return err
		}

		// Validate JWT token
		claims, err := jwtService.ValidateAccessToken(spanCtx, accessToken)
		if err != nil {
			logger.WithError(err).Error("access token is invalid or expired")
			return errcode.ErrTokenIsExpired
		}

		// Store claims in locals
		c.Locals(authKey, claims)
		return c.Next()
	}
}

// GetUser retrieves user claims from fiber context with type assertion
func GetUser(ctx *fiber.Ctx) *service.Claims {
	return ctx.Locals(authKey).(*service.Claims)
}
