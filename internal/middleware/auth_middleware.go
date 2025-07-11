package middleware

import (
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errcode"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func AuthMiddleware(jwtService *service.JwtService, blacklistService *service.BlacklistService, log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Error("authorization header required")
			return errcode.ErrAuthorizationHeader
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("bearer not found in Authorization header")
			return errcode.ErrBearerHeader
		}

		accessToken := strings.TrimPrefix(authHeader, "Bearer ")
		if accessToken == "" {
			log.Warn("access token not found in Authorization header")
			return errcode.ErrAccessTokenMissing
		}

		if err := blacklistService.IsTokenBlacklisted(accessToken); err != nil {
			log.Warn("already logout")
			return err
		}

		claims, err := jwtService.ValidateAccessToken(accessToken)
		if err != nil {
			log.WithError(err).Error("token is expired")
			return errcode.ErrTokenIsExpired
		}

		c.Locals("auth", claims)
		return c.Next()
	}
}

func GetUser(ctx *fiber.Ctx) *service.Claims {
	return ctx.Locals("auth").(*service.Claims)
}
