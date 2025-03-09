package logutil

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// RequestEntry logs the entry of a request with standard fields
func AccessLog(logger *logrus.Logger, ctx *fiber.Ctx, method string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"method":    method,
		"path":      ctx.Path(),
		"client_ip": ctx.IP(),
	})
}
