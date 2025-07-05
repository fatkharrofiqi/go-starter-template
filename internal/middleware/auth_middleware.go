package middleware

import (
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/apperrors"
	"go-starter-template/internal/utils/jwtutil"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func AuthMiddleware(secret string, redis *redis.Client, log *logrus.Logger, blacklist repository.TokenBlacklistRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Error("authorization header required")
			return apperrors.ErrAuthorizationHeader
		}

		tokenString := strings.Split(authHeader, "Bearer ")[1]
		if _, err := blacklist.IsBlacklisted(tokenString); err != nil {
			log.Error("token is blacklisted")
			return apperrors.ErrTokenBlacklisted
		}

		claims, err := jwtutil.ValidateToken(tokenString, secret)
		if err != nil {
			log.WithError(err).Error("invalid token")
			return apperrors.ErrTokenIsExpired
		}

		c.Locals("auth", claims)
		return c.Next()
	}
}

func GetUser(ctx *fiber.Ctx) *jwtutil.Claims {
	return ctx.Locals("auth").(*jwtutil.Claims)
}
