package logutil

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestEntry logs the entry of a request with standard fields
func RequestEntry(logger *logrus.Logger, ctx *gin.Context, method string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"method":    method,
		"path":      ctx.Request.URL.Path,
		"client_ip": ctx.ClientIP(),
	})
}

// Error logs an error with additional context, handling nil errors gracefully
func Error(logger *logrus.Logger, message string, err error, fields ...logrus.Fields) {
	entry := logger.WithFields(logrus.Fields{})
	if err != nil {
		entry = entry.WithField("error", err.Error())
	}
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Warn(message)
}
