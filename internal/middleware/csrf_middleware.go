package middleware

import (
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/apperrors"
	"go-starter-template/internal/utils/csrfutil"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func CsrfMiddleware(secret string, log *logrus.Logger, blacklist *repository.TokenBlacklist) fiber.Handler {
	return func(c *fiber.Ctx) error {
		csrfToken := c.Get("X-CSRF-Token")
		if csrfToken == "" {
			log.Error("access token is required")
			return apperrors.ErrCsrfTokenHeader
		}

		if blacklist.IsBlacklisted(csrfToken) {
			log.Error("access token is already used")
			return apperrors.ErrTokenBlacklisted
		}

		claims, err := csrfutil.ValidateToken(csrfToken, secret)
		if err != nil {
			log.WithError(err).Error("invalid token")
			return apperrors.ErrCsrfTokenIsExpired
		}

		if claims.Path != c.Path() {
			return apperrors.ErrCsrfTokenInvalidPath
		}

		blacklist.Add(csrfToken)

		return c.Next()
	}
}
