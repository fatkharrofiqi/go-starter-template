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
    bearerKeyword = "Bearer"
    bearerLen     = len(bearerKeyword)
    authKey       = "auth"
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

        // Check prefix first (be lenient: allow "Bearer" without trailing space)
        if !strings.HasPrefix(authHeader, bearerKeyword) {
            logger.Warn("invalid authorization header format")
            return errcode.ErrBearerHeader
        }

        // Extract token after the "Bearer" keyword and trim spaces
        accessToken := strings.TrimSpace(authHeader[bearerLen:])
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
