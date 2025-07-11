package middleware

import (
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errcode"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func CsrfMiddleware(csrfService *service.CsrfService, blacklistService *service.BlacklistService, log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		csrfToken := c.Get("X-CSRF-Token")
		if csrfToken == "" {
			log.Error("csrf token is required")
			return errcode.ErrCsrfTokenHeader
		}

		if err := blacklistService.IsTokenBlacklisted(csrfToken); err != nil {
			log.Error("csrf token is already used")
			return errcode.ErrTokenBlacklisted
		}

		claims, err := csrfService.ValidateCsrfToken(csrfToken)
		if err != nil {
			log.WithError(err).Error("invalid token")
			return errcode.ErrCsrfTokenIsExpired
		}

		if claims.Path != c.Path() {
			log.WithError(err).Error("csrf token is invalid for this url")
			return errcode.ErrCsrfTokenInvalidPath
		}

		if err := blacklistService.Add(csrfToken); err != nil {
			log.WithError(err).Error("can't blacklist the csrf token")
			return errcode.ErrCantBlacklistToken
		}

		return c.Next()
	}
}
