package middleware

import (
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

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.WithContext(spanCtx).Error("authorization header required")
			return errcode.ErrAuthorizationHeader
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.WithContext(spanCtx).Warn("bearer not found in Authorization header")
			return errcode.ErrBearerHeader
		}

		accessToken := strings.TrimPrefix(authHeader, "Bearer ")
		if accessToken == "" {
			log.WithContext(spanCtx).Warn("access token not found in Authorization header")
			return errcode.ErrAccessTokenMissing
		}

		if err := blacklistService.IsTokenBlacklisted(spanCtx, accessToken); err != nil {
			log.WithContext(spanCtx).Warn("already logout")
			return err
		}

		claims, err := jwtService.ValidateAccessToken(spanCtx, accessToken)
		if err != nil {
			log.WithContext(spanCtx).WithError(err).Error("token is expired")
			return errcode.ErrTokenIsExpired
		}

		c.Locals("auth", claims)
		return c.Next()
	}
}

func GetUser(ctx *fiber.Ctx) *service.Claims {
	return ctx.Locals("auth").(*service.Claims)
}
