package config

import (
	"go-starter-template/internal/config/env"

	"github.com/sirupsen/logrus"
)

func NewLogger(config *env.Config) *logrus.Logger {
	log := logrus.New()

	log.SetLevel(logrus.Level(config.Log.Level))
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	return log
}
