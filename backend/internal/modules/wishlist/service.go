package wishlist

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Add saves a public product in the caller's wishlist.
//
// No transaction: the check and the insert are two statements, and the
// product could be archived between them. That is harmless, because the list
// only shows public products. Two concurrent adds of the same product are
// settled by the unique index (ON CONFLICT DO NOTHING), not by a lock.
func (s *Service) Add(ctx context.Context, userID, productID uuid.UUID) error {
	repo := NewRepository(s.pool)
	public, err := repo.IsPublicProduct(ctx, productID)
	if err != nil {
		return err
	}
	if !public {
		return ErrProductNotFound
	}
	return repo.Add(ctx, userID, productID)
}

// Remove deletes a product from the caller's wishlist. Removing a product
// that is not there (or not public anymore) is not an error.
func (s *Service) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	return NewRepository(s.pool).Remove(ctx, userID, productID)
}

// List returns one page of the caller's wishlist, newest first.
func (s *Service) List(ctx context.Context, userID uuid.UUID, page, pageSize int) (Page, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	errs := validation.Errors{}
	if page < 1 || page > 10_000 {
		errs.Add("page", "must be between 1 and 10000")
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		errs.Add("pageSize", "must be between 1 and 48")
	}
	if err := errs.Err(); err != nil {
		return Page{}, err
	}

	repo := NewRepository(s.pool)
	items, total, err := repo.List(ctx, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page{}, err
	}
	// Past the last page there are no rows and so no total; ask for the
	// first row just to learn it.
	if len(items) == 0 && page > 1 {
		if _, total, err = repo.List(ctx, userID, 1, 0); err != nil {
			return Page{}, err
		}
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}
	return Page{Items: items, Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages}, nil
}

// ProductIDs returns the ids of the caller's wishlisted public products.
func (s *Service) ProductIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return NewRepository(s.pool).ProductIDs(ctx, userID)
}
