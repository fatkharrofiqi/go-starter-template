package repository

import (
    "context"
    "database/sql"
)

// UnitOfWork encapsulates transaction boundaries and exposes repositories
// bound to the active transaction.
type UnitOfWork struct {
    db *sql.DB
}

func NewUnitOfWork(db *sql.DB) *UnitOfWork {
    return &UnitOfWork{db: db}
}

// Do runs fn within a transaction. It begins the transaction, injects it
// into the context so repositories pick it up via getExecutor, and commits
// or rolls back based on fn's result.
func (u *UnitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error) error {
    tx, err := u.db.BeginTx(ctx, &sql.TxOptions{})
    if err != nil {
        return err
    }
    ctxWithTx := context.WithValue(ctx, TxKey, tx)

    if err := fn(ctxWithTx); err != nil {
        _ = tx.Rollback()
        return err
    }
    return tx.Commit()
}