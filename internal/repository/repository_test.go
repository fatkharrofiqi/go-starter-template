package repository

import (
    "context"
    "database/sql"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
)

func TestRepository_getExecutor(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    repo := &Repository{db: db}

    type testcase struct {
        name       string
        setCtx     func(ctx context.Context) context.Context
        expectIsTx bool
    }

    cases := []testcase{
        {
            name: "NoTxInContext",
            setCtx: func(ctx context.Context) context.Context {
                return ctx
            },
            expectIsTx: false,
        },
        {
            name: "WithNilTxInContext",
            setCtx: func(ctx context.Context) context.Context {
                return context.WithValue(ctx, TxKey, (*sql.Tx)(nil))
            },
            expectIsTx: false,
        },
        {
            name: "WithNonTxValueInContext",
            setCtx: func(ctx context.Context) context.Context {
                return context.WithValue(ctx, TxKey, "not-a-tx")
            },
            expectIsTx: false,
        },
        {
            name: "WithTxInContext",
            setCtx: func(ctx context.Context) context.Context {
                mock.ExpectBegin()
                tx, _ := db.BeginTx(ctx, nil)
                return context.WithValue(ctx, TxKey, tx)
            },
            expectIsTx: true,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            base := context.Background()
            ctx := tc.setCtx(base)
            exec := repo.getExecutor(ctx)

            if tc.expectIsTx {
                tx, ok := exec.(*sql.Tx)
                if !ok || tx == nil {
                    t.Fatalf("expected executor to be *sql.Tx, got %T", exec)
                }
            } else {
                gotDB, ok := exec.(*sql.DB)
                if !ok || gotDB != db {
                    t.Fatalf("expected executor to be repo.db (*sql.DB), got %T", exec)
                }
            }

            if err := mock.ExpectationsWereMet(); err != nil {
                t.Fatalf("there were unmet expectations: %v", err)
            }
        })
    }
}