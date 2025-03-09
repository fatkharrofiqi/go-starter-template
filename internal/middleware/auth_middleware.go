package middleware

import (
	"go-starter-template/internal/utils/jwtutil"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func AuthMiddleware(secret string, log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Error("authorization header required")
			return fiber.NewError(fiber.ErrUnauthorized.Code, "authorization header required")
		}

		tokenString := strings.Split(authHeader, "Bearer ")[1]
		claims, err := jwtutil.ValidateToken(tokenString, secret)
		if err != nil {
			log.WithError(err).Error("invalid token")
			return fiber.NewError(fiber.ErrUnauthorized.Code, err.Error())
		}

		log.Debugf("User : %v", claims)
		c.Locals("auth", claims)
		return c.Next()
	}
}

func GetUser(ctx *fiber.Ctx) *jwtutil.Claims {
	return ctx.Locals("auth").(*jwtutil.Claims)
}
