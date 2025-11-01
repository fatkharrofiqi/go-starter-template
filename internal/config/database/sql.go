package database

import (
	"database/sql"
	"go-starter-template/internal/config/env"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// NewSQLDatabase initializes and returns a database/sql connection using pgx driver.
func NewSQLDatabase(log *logrus.Logger, config *env.Config) *sql.DB {
	dsn := config.Database.DSN

	// Instrument database/sql with OpenTelemetry so queries are captured
	sqlDB, err := otelsql.Open("pgx", dsn,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	)
	if err != nil {
		log.Fatalf("failed to open sql database: %v", err)
	}

	idleConnection := config.Database.Pool.Idle
	maxConnection := config.Database.Pool.Max
	maxLifeTimeConnection := config.Database.Pool.Lifetime

	sqlDB.SetMaxIdleConns(idleConnection)
	sqlDB.SetMaxOpenConns(maxConnection)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifeTimeConnection) * time.Second)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("failed to ping sql database: %v", err)
	}

	log.Info("SQL database connection established successfully")
	return sqlDB
}