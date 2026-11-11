package health

import (
	"context"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

// Repository is the only layer that talks to PostgreSQL. Handlers never see
// the pool and never write SQL; they ask the repository for what they need.
//
// The SQL itself lives in db/queries/health.sql. sqlc turns it into the
// queries.Queries.Ping method, so a typo in a column name or a wrong Go type
// is caught by `make sqlc` / the compiler, not at runtime.
type Repository struct {
	q *queries.Queries
}

// NewRepository accepts queries.DBTX, which both *pgxpool.Pool and pgx.Tx
// satisfy. Later modules use the same trick to run repository methods inside
// a transaction (checkout!) without changing the repository code.
func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

// Ping runs a real query so readiness proves the full path works:
// borrow a connection from the pool -> send SQL -> read a row -> return the
// connection. pool.Ping alone would skip the "read a row" part.
func (r *Repository) Ping(ctx context.Context) error {
	_, err := r.q.Ping(ctx)
	return err
}
