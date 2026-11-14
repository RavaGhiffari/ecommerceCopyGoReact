package orders

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	DefaultPageSize = 10
	MaxPageSize     = 50
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ListForBuyer returns one page of the buyer's orders, newest first.
// page and pageSize 0 mean "use the default".
func (s *Service) ListForBuyer(ctx context.Context, buyerID uuid.UUID, page, pageSize int) (Page, error) {
	errs := validation.Errors{}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	if page < 1 {
		errs.Add("page", "must be 1 or more")
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		errs.Add("pageSize", fmt.Sprintf("must be between 1 and %d", MaxPageSize))
	}
	if err := errs.Err(); err != nil {
		return Page{}, err
	}

	repo := NewRepository(s.pool)
	total, err := repo.CountForBuyer(ctx, buyerID)
	if err != nil {
		return Page{}, err
	}
	items, err := repo.ListForBuyer(ctx, buyerID, int32(pageSize), int32((page-1)*pageSize))
	if err != nil {
		return Page{}, err
	}
	return Page{
		Items: items, Page: page, PageSize: pageSize, Total: total,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
}

// GetForBuyer returns ErrNotFound for an order of another buyer: the caller
// cannot tell "not yours" from "does not exist".
func (s *Service) GetForBuyer(ctx context.Context, buyerID, orderID uuid.UUID) (Order, error) {
	return NewRepository(s.pool).GetForBuyer(ctx, buyerID, orderID)
}
