package test

import (
	"sync"
	"testing"
	"time"

	"go-starter-template/internal/config/database"
	"go-starter-template/internal/config/env"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewDatabase(t *testing.T) {
	config := env.NewConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	db := database.NewDatabase(logger, config)
	assert.NotNil(t, db, "Database instance should not be nil")

	sqlDB, err := db.DB()
	assert.NoError(t, err, "Getting SQL DB should not return an error")
	defer sqlDB.Close()

	// Log initial pool stats
	stats := sqlDB.Stats()
	logger.Infof("Initial - MaxOpen: %d, Idle: %d, InUse: %d",
		stats.MaxOpenConnections, stats.Idle, stats.InUse)
	assert.Equal(t, 100, stats.MaxOpenConnections, "Max open connections should be 100")
	assert.GreaterOrEqual(t, stats.Idle, 1, "Initial idle should be at least 1")

	// Simulate concurrent load to increase idle connections
	var wg sync.WaitGroup
	numGoroutines := 15 // More than idle limit (10) to force pool growth
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := sqlDB.Ping()
			assert.NoError(t, err, "Ping should succeed")
			time.Sleep(50 * time.Millisecond) // Hold connection briefly to keep it in use
		}()
	}
	wg.Wait()

	// Check stats after concurrent load
	stats = sqlDB.Stats()
	logger.Infof("After load - MaxOpen: %d, Idle: %d, InUse: %d",
		stats.MaxOpenConnections, stats.Idle, stats.InUse)
	assert.GreaterOrEqual(t, stats.Idle, 2, "Idle connections should increase after concurrent load")
	assert.LessOrEqual(t, stats.Idle, 10, "Idle should not exceed configured max idle (10)")
}
