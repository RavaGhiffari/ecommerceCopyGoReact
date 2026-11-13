// Package database creates the PostgreSQL connection pool.
//
// Why a pool: opening a PostgreSQL connection is expensive (TCP + TLS + auth),
// and each HTTP request needs one only for a few milliseconds. pgxpool keeps a
// set of open connections and lends one to each query, returning it after.
// The whole application shares ONE pool, created once at startup in main.go.
package database

import (
	"context"
	"fmt"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
)

// NewPool builds the pool and verifies the database is reachable.
// If PostgreSQL is down or the credentials are wrong, this returns an error
// and the API refuses to start instead of failing on the first request.
func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		// Don't wrap err with the URL: it contains the password.
		return nil, fmt.Errorf("parse DATABASE_URL: invalid connection string")
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	// Teach every new connection to read/write NUMERIC as decimal.Decimal
	// (exact money arithmetic) instead of float64 or pgtype.Numeric.
	poolCfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// NewWithConfig connects lazily; Ping forces one real round trip now.
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
