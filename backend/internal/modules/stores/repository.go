package stores

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

type Repository struct {
	q *queries.Queries
}

func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

// The store queries select every column, so sqlc returns its queries.Store model.
func toStore(r queries.Store) Store {
	return Store{
		ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, Slug: r.Slug, Description: r.Description,
		Location: r.Location, Status: r.Status, CreatedAt: r.CreatedAt,
	}
}

func (r *Repository) Create(ctx context.Context, s Store) (Store, error) {
	row, err := r.q.CreateStore(ctx, queries.CreateStoreParams{
		OwnerID: s.OwnerID, Name: s.Name, Slug: s.Slug, Description: s.Description, Location: s.Location,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "stores_owner_id_key":
				return Store{}, ErrAlreadyHasStore
			case "stores_slug_key":
				return Store{}, ErrSlugTaken
			}
		}
		return Store{}, fmt.Errorf("create store: %w", err)
	}
	return toStore(row), nil
}

func (r *Repository) GetByOwner(ctx context.Context, ownerID uuid.UUID) (Store, error) {
	row, err := r.q.GetStoreByOwner(ctx, ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Store{}, ErrNotFound
	}
	if err != nil {
		return Store{}, fmt.Errorf("get store by owner: %w", err)
	}
	return toStore(row), nil
}

// GetActiveBySlug is the public lookup: suspended stores are "not found".
func (r *Repository) GetActiveBySlug(ctx context.Context, slug string) (Store, error) {
	row, err := r.q.GetActiveStoreBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Store{}, ErrNotFound
	}
	if err != nil {
		return Store{}, fmt.Errorf("get store by slug: %w", err)
	}
	s := Store{
		ID: row.ID, OwnerID: row.OwnerID, Name: row.Name, Slug: row.Slug, Description: row.Description,
		Location: row.Location, Status: row.Status, CreatedAt: row.CreatedAt,
	}
	s.ActiveProductCount = row.ActiveProductCount
	s.RatingCount, s.RatingTotal = row.RatingCount, row.RatingTotal
	return s, nil
}

func (r *Repository) Update(ctx context.Context, s Store) (Store, error) {
	row, err := r.q.UpdateStore(ctx, queries.UpdateStoreParams{
		ID: s.ID, Name: s.Name, Description: s.Description, Location: s.Location,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Store{}, ErrNotFound
	}
	if err != nil {
		return Store{}, fmt.Errorf("update store: %w", err)
	}
	return toStore(row), nil
}
