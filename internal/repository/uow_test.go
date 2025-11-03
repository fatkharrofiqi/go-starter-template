package repository

import (
    "context"
    "errors"
    "regexp"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
)

func TestUnitOfWork_Do(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    uow := NewUnitOfWork(db)

    type testcase struct {
        name      string
        setupMock func()
        fn        func(ctx context.Context) error
        expectErr string
    }

    cases := []testcase{
        {
            name: "Success_Commits",
            setupMock: func() {
                mock.ExpectBegin()
                mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = name WHERE id = $1")).
                    WithArgs(1).
                    WillReturnResult(sqlmock.NewResult(0, 1))
                mock.ExpectCommit()
            },
            fn: func(ctx context.Context) error {
                // Ensure repositories pick tx from context via getExecutor by running a statement
                repo := &Repository{db: db}
                exec := repo.getExecutor(ctx)
                if _, err := exec.ExecContext(ctx, "UPDATE users SET name = name WHERE id = $1", 1); err != nil {
                    return err
                }
                return nil
            },
            expectErr: "",
        },
        {
            name: "FnError_RollsBack",
            setupMock: func() {
                mock.ExpectBegin()
                mock.ExpectRollback()
            },
            fn: func(ctx context.Context) error {
                return errors.New("fn failed")
            },
            expectErr: "fn failed",
        },
        {
            name: "BeginTxError",
            setupMock: func() {
                mock.ExpectBegin().WillReturnError(errors.New("begin error"))
            },
            fn: func(ctx context.Context) error {
                // should never be called
                t.Fatalf("fn should not be called on begin error")
                return nil
            },
            expectErr: "begin error",
        },
        {
            name: "CommitError_Propagates",
            setupMock: func() {
                mock.ExpectBegin()
                mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = name WHERE id = $1")).
                    WithArgs(2).
                    WillReturnResult(sqlmock.NewResult(0, 1))
                mock.ExpectCommit().WillReturnError(errors.New("commit error"))
            },
            fn: func(ctx context.Context) error {
                repo := &Repository{db: db}
                exec := repo.getExecutor(ctx)
                if _, err := exec.ExecContext(ctx, "UPDATE users SET name = name WHERE id = $1", 2); err != nil {
                    return err
                }
                return nil
            },
            expectErr: "commit error",
        },
        {
            name: "RollbackError_Ignored",
            setupMock: func() {
                mock.ExpectBegin()
                mock.ExpectRollback().WillReturnError(errors.New("rollback error"))
            },
            fn: func(ctx context.Context) error {
                return errors.New("fn failed")
            },
            expectErr: "fn failed",
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            tc.setupMock()
            err := uow.Do(context.Background(), tc.fn)
            if tc.expectErr == "" && err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if tc.expectErr != "" {
                if err == nil || err.Error() != tc.expectErr {
                    t.Fatalf("expected error %q, got %v", tc.expectErr, err)
                }
            }

            if err := mock.ExpectationsWereMet(); err != nil {
                t.Fatalf("unmet expectations: %v", err)
            }
        })
    }
}