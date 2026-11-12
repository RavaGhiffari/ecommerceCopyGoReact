package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx runs fn inside one PostgreSQL transaction: either everything fn did
// is committed, or (if fn returns an error or panics) nothing is.
//
// Usage from a service:
//
//	err := database.WithTx(ctx, pool, func(tx pgx.Tx) error {
//	    repo := users.NewRepository(tx) // same repository code, now inside the tx
//	    ...
//	})
//
// The service decides WHICH steps belong together (a business rule); the
// repositories stay unaware of transactions because pgx.Tx and *pgxpool.Pool
// both satisfy queries.DBTX.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// Rollback after a successful Commit is a no-op, so this is always safe and
	// guarantees cleanup on every early return or panic.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
