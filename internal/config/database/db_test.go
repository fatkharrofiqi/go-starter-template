package database

import (
    "database/sql"
    "errors"
    "testing"

    "go-starter-template/internal/config/env"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/sirupsen/logrus"
    "github.com/uptrace/opentelemetry-go-extra/otelsql"
)

// helper to build minimal config
func testConfig() *env.Config {
	cfg := &env.Config{}
	cfg.Database.DSN = "test-dsn"
	cfg.Database.Pool.Idle = 1
	cfg.Database.Pool.Max = 2
	cfg.Database.Pool.Lifetime = 1 // seconds
	return cfg
}

func TestNewDatabase_Success(t *testing.T) {
    // arrange: sqlOpen returns sqlmock DB, Ping succeeds
    db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()
    mock.ExpectPing()

    // override seam
    orig := sqlOpen
    sqlOpen = func(driverName, dsn string, opts ...otelsql.Option) (*sql.DB, error) {
        return db, nil
    }
    defer func() { sqlOpen = orig }()

	log := logrus.New()
	cfg := testConfig()

	// act
	got := NewDatabase(log, cfg)

	// assert
	if got == nil {
		t.Fatalf("expected *sql.DB, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNewDatabase_OpenError_Fatal(t *testing.T) {
    log := logrus.New()
    // capture fatal exit without terminating process
    exitCalled := false
    log.ExitFunc = func(code int) { exitCalled = true; panic("exit") }

    cfg := testConfig()

    // override seam to return open error
    orig := sqlOpen
    sqlOpen = func(driverName, dsn string, opts ...otelsql.Option) (*sql.DB, error) {
        return nil, errors.New("open failed")
    }
    defer func() { sqlOpen = orig }()

	// act
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic from ExitFunc, got none")
		}
		if !exitCalled {
			t.Fatalf("expected ExitFunc to be called")
		}
	}()
	_ = NewDatabase(log, cfg)
}

func TestNewDatabase_PingError_Fatal(t *testing.T) {
    // arrange: sqlOpen returns sqlmock DB, Ping fails
    db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()
    mock.ExpectPing().WillReturnError(errors.New("ping failed"))

    // override seam
    orig := sqlOpen
    sqlOpen = func(driverName, dsn string, opts ...otelsql.Option) (*sql.DB, error) {
        return db, nil
    }
    defer func() { sqlOpen = orig }()

	log := logrus.New()
	exitCalled := false
	log.ExitFunc = func(code int) { exitCalled = true; panic("exit") }

	cfg := testConfig()

	// act
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic from ExitFunc, got none")
		}
		if !exitCalled {
			t.Fatalf("expected ExitFunc to be called")
		}
		// ensure expectations met (Ping attempted)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	}()
	_ = NewDatabase(log, cfg)
}