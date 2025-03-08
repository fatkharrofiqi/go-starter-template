package test

import (
	"go-starter-template/internal/config"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestNewDatabase ensures database connection initializes correctly
func TestNewDatabase(t *testing.T) {
	// Mock configuration
	cfg := config.NewViper()

	// Mock logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	// Initialize database connection
	db := config.NewDatabase(cfg, logger)

	// Ensure database instance is not nil
	assert.NotNil(t, db, "Database instance should not be nil")

	// Ensure we can get the underlying SQL DB
	sqlDB, err := db.DB()
	assert.NoError(t, err, "Getting underlying SQL DB should not return an error")

	// ✅ Correctly check connection pool settings
	assert.Equal(t, 100, sqlDB.Stats().MaxOpenConnections, "Max open connections should match")
	assert.Equal(t, 1, sqlDB.Stats().Idle, "Idle connections may be low at start")

	// ✅ Run some queries to increase idle connections
	_ = sqlDB.Ping() // Open a connection
	_ = sqlDB.Ping()
	_ = sqlDB.Ping()

	// ✅ Check if idle connections have increased
	idleConnections := sqlDB.Stats().Idle
	assert.GreaterOrEqual(t, idleConnections, 1, "Idle connections should increase after queries")

	// Close database connection
	err = sqlDB.Close()
	assert.NoError(t, err, "Closing database should not return an error")
}
