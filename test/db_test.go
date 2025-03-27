package test

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"go-starter-template/internal/config/env"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupDB membuat koneksi database dengan konfigurasi tertentu untuk benchmark
func setupDB(idle, max int, lifetime time.Duration) *sql.DB {
	cfg := env.NewConfig() // Asumsi ini mengambil config.yml
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{
		Logger: logger.New(log, logger.Config{
			SlowThreshold: time.Second * 5,
			LogLevel:      logger.Info,
		}),
	})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}

	sqlDB.SetMaxIdleConns(idle)
	sqlDB.SetMaxOpenConns(max)
	sqlDB.SetConnMaxLifetime(lifetime)

	return sqlDB
}

// BenchmarkDBPing mengukur performa Ping dengan konfigurasi tertentu
func BenchmarkDBPing(b *testing.B) {
	configs := []struct {
		name     string
		idle     int
		max      int
		lifetime time.Duration
	}{
		{"Idle5_Max300_Lifetime60s", 5, 100, 60 * time.Second},
		{"Idle10_Max300_Lifetime60s", 10, 100, 60 * time.Second}, // Config aslimu
		{"Idle20_Max300_Lifetime60s", 20, 100, 60 * time.Second},
		{"Idle30_Max300_Lifetime60s", 30, 100, 60 * time.Second},
		{"Idle40_Max300_Lifetime60s", 40, 100, 60 * time.Second},
		{"Idle50_Max300_Lifetime60s", 50, 100, 60 * time.Second},
		{"Idle60_Max300_Lifetime60s", 60, 100, 60 * time.Second},
		{"Idle70_Max300_Lifetime60s", 70, 100, 60 * time.Second},
		{"Idle80_Max300_Lifetime60s", 80, 100, 60 * time.Second},
		{"Idle90_Max300_Lifetime60s", 90, 100, 60 * time.Second},
		{"Idle100_Max300_Lifetime60s", 100, 100, 60 * time.Second},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			db := setupDB(cfg.idle, cfg.max, cfg.lifetime)
			defer db.Close()

			// Reset timer untuk mengabaikan waktu setup
			b.ResetTimer()

			// Jalankan benchmark dengan konkurensi
			var wg sync.WaitGroup
			numWorkers := 80 // Simulasi 50 pengguna bersamaan
			for i := 0; i < b.N; i++ {
				wg.Add(numWorkers)
				for j := 0; j < numWorkers; j++ {
					go func() {
						defer wg.Done()
						_ = db.Ping() // Operasi yang diukur
					}()
				}
				wg.Wait()
			}
		})
	}
}
