package config

import (
	"go-starter-template/internal/config/env"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDatabase initializes and returns a PostgreSQL database connection
func NewDatabase(config *env.Config, log *logrus.Logger) *gorm.DB {
	dsn := config.Database.DSN

	idleConnection := config.Database.Pool.Idle
	maxConnection := config.Database.Pool.Max
	maxLifeTimeConnection := config.Database.Pool.Lifetime

	// Initialize GORM with PostgreSQL driver
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(log, logger.Config{
			SlowThreshold:             time.Second * 5,
			Colorful:                  true,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			LogLevel:                  logger.Info,
		}),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Get the underlying SQL connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get database instance: %v", err)
	}

	// Configure connection pooling
	sqlDB.SetMaxIdleConns(idleConnection)
	sqlDB.SetMaxOpenConns(maxConnection)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifeTimeConnection) * time.Second)

	log.Info("Database connection established successfully")
	return db
}
