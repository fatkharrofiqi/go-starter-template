package config

import (
	"go-starter-template/internal/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	logrus.Info("Starting database migrations")

	if err := db.AutoMigrate(&model.User{}); err != nil {
		logrus.WithError(err).Error("Failed to run migrations")
		return err
	}

	logrus.Info("Database migrations completed successfully")
	return nil
}
