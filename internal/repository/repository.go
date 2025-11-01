package repository

import (
	"context"
	"database/sql"
)

type contextKey string

var TxKey contextKey = "tx"

// SQLExecutor is implemented by both *sql.DB and *sql.Tx
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Repository struct {
	db *sql.DB
}

// getExecutor returns an executor that is either the active *sql.Tx (if present in ctx)
// or the base *sql.DB. Use this when calling ExecContext/QueryContext methods.
func (r *Repository) getExecutor(ctx context.Context) SQLExecutor {
	if tx, ok := ctx.Value(TxKey).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return r.db
}
