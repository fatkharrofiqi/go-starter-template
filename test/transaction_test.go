package test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"go-starter-template/internal/model"
	"go-starter-template/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUnitOfWork_Commit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserRepository(db)
	uow := repository.NewUnitOfWork(db)

	ctx := context.Background()
	email := fmt.Sprintf("tx_commit_%d@test.com", time.Now().UnixNano())
	uid := uuid.NewString()

	// Expectations: begin, count=0, insert, commit, then count=1
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(regexp.QuoteMeta("\n        INSERT INTO users (uuid, name, email, password, created_at, updated_at)\n        VALUES ($1, $2, $3, $4, NOW(), NOW())\n    ")).
		WithArgs(uid, "Tx Commit", email, "secret").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = uow.Do(ctx, func(txCtx context.Context) error {
		count, err := repo.CountByEmail(txCtx, email)
		if err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("email already exists in precondition")
		}
		user := &model.User{UUID: uid, Name: "Tx Commit", Email: email, Password: "secret"}
		return repo.Create(txCtx, user)
	})
	require.NoError(t, err, "transaction commit should not error")

	// After commit, verify count=1
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	total, err := repo.CountByEmail(ctx, email)
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "user should exist after commit")

	// Ensure all expectations were met
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUnitOfWork_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserRepository(db)
	uow := repository.NewUnitOfWork(db)

	ctx := context.Background()
	email := fmt.Sprintf("tx_rollback_%d@test.com", time.Now().UnixNano())
	uid := uuid.NewString()

	// Expectations: begin, insert, rollback, then count=0
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("\n        INSERT INTO users (uuid, name, email, password, created_at, updated_at)\n        VALUES ($1, $2, $3, $4, NOW(), NOW())\n    ")).
		WithArgs(uid, "Tx Rollback", email, "secret").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()

	err = uow.Do(ctx, func(txCtx context.Context) error {
		user := &model.User{UUID: uid, Name: "Tx Rollback", Email: email, Password: "secret"}
		if err := repo.Create(txCtx, user); err != nil {
			return err
		}
		// force rollback
		return fmt.Errorf("force rollback")
	})
	require.Error(t, err, "transaction should return error to trigger rollback")

	// After rollback, verify count=0
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	total, err := repo.CountByEmail(ctx, email)
	require.NoError(t, err)
	require.Equal(t, int64(0), total, "user should not exist after rollback")

	// Ensure all expectations were met
	require.NoError(t, mock.ExpectationsWereMet())
}
