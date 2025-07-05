package middleware

import (
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/apperrors"
	"go-starter-template/internal/utils/csrfutil"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func CsrfMiddleware(secret string, log *logrus.Logger, blacklist repository.TokenBlacklistRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		csrfToken := c.Get("X-CSRF-Token")
		if csrfToken == "" {
			log.Error("csrf token is required")
			return apperrors.ErrCsrfTokenHeader
		}

		if _, err := blacklist.IsBlacklisted(csrfToken); err != nil {
			log.Error("csrf token is already used")
			return apperrors.ErrTokenBlacklisted
		}

		claims, err := csrfutil.ValidateToken(csrfToken, secret)
		if err != nil {
			log.WithError(err).Error("invalid token")
			return apperrors.ErrCsrfTokenIsExpired
		}

		if claims.Path != c.Path() {
			log.WithError(err).Error("csrf token is invalid for this url")
			return apperrors.ErrCsrfTokenInvalidPath
		}

		if err := blacklist.Add(csrfToken); err != nil {
			log.WithError(err).Error("can't blacklist the csrf token")
			return apperrors.ErrCantBlacklistToken
		}

		return c.Next()
	}
}
