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

func AuthMiddleware(jwtService *service.JwtService, blacklistService *service.BlacklistService, log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tracer := otel.Tracer("AuthMiddleware")
		spanCtx, span := tracer.Start(c.UserContext(), "AuthMiddleware")
		defer span.End()

		logger := log.WithContext(spanCtx)

		authHeader := c.Get("Authorization")
		if len(authHeader) < 8 || !strings.HasPrefix(authHeader, "Bearer ") {
			if authHeader == "" {
				logger.Error("authorization header missing")
				return errcode.ErrAuthorizationHeader
			}
			logger.Warn("invalid authorization header format")
			return errcode.ErrBearerHeader
		}

		accessToken := authHeader[7:] // TrimPrefix without allocation
		if accessToken == "" {
			logger.Warn("access token missing in header")
			return errcode.ErrAccessTokenMissing
		}

		if err := blacklistService.IsTokenBlacklisted(spanCtx, accessToken, constant.TokenTypeAccess); err != nil {
			logger.Warn("access token is blacklisted")
			return err
		}

		claims, err := jwtService.ValidateAccessToken(spanCtx, accessToken)
		if err != nil {
			logger.WithError(err).Error("access token is invalid or expired")
			return errcode.ErrTokenIsExpired
		}

		c.Locals("auth", claims)
		return c.Next()
	}
}

func GetUser(ctx *fiber.Ctx) *service.Claims {
	return ctx.Locals("auth").(*service.Claims)
}
