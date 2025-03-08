package config

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDatabase initializes and returns a PostgreSQL database connection
func NewDatabase(viper *viper.Viper, log *logrus.Logger) *gorm.DB {
	dsn := viper.GetString("database.dsn")

	idleConnection := viper.GetInt("database.pool.idle")
	maxConnection := viper.GetInt("database.pool.max")
	maxLifeTimeConnection := viper.GetInt("database.pool.lifetime")

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
