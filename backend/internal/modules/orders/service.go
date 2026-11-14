package orders

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/inventory"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// ReferenceOrder is inventory_movements.reference_type for a refund restock.
const ReferenceOrder = "order"

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
	page, pageSize = paging(errs, page, pageSize)
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
	return newPage(items, page, pageSize, total), nil
}

// paging applies the defaults (0 = default) and records invalid values.
func paging(errs validation.Errors, page, pageSize int) (int, int) {
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
	return page, pageSize
}

func newPage(items []Order, page, pageSize int, total int64) Page {
	return Page{
		Items: items, Page: page, PageSize: pageSize, Total: total,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}
}

// GetForBuyer returns ErrNotFound for an order of another buyer: the caller
// cannot tell "not yours" from "does not exist".
func (s *Service) GetForBuyer(ctx context.Context, buyerID, orderID uuid.UUID) (Order, error) {
	return NewRepository(s.pool).GetForBuyer(ctx, buyerID, orderID)
}

// CancelForBuyer cancels one PAID order before the seller starts processing
// it: the order becomes refunded (simulated; no money moves) and its stock
// comes back (decision Q8). One transaction:
//
//  1. Lock the order row, filtered by buyer (someone else's -> 404). The
//     seller's "processing" (5d) locks the same row, so "buyer cancels" and
//     "seller starts processing" cannot both win; the second one sees the
//     new status and gets a 409.
//  2. Unpaid -> UnpaidError: the whole checkout is cancelled instead.
//  3. paid -> refunded through the transition table (anything else -> 409).
//  4. Lock the order's inventory rows in variant order, restock each line
//     (+ ledger rows referencing the order) and lower sold_count.
func (s *Service) CancelForBuyer(ctx context.Context, buyerID, orderID uuid.UUID) (Order, error) {
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)

		// 1.
		o, err := repo.LockForBuyer(ctx, buyerID, orderID)
		if err != nil {
			return err
		}
		// 2.
		if o.Status == StatusPendingPayment {
			return &UnpaidError{CheckoutID: o.CheckoutID.String()}
		}
		// 3.
		if err := repo.Transition(ctx, &o, StatusRefunded, ByBuyer, &buyerID, "Cancelled by the buyer; refund simulated"); err != nil {
			return err
		}
		// 4.
		lines, err := repo.StockForOrder(ctx, o.ID)
		if err != nil {
			return err
		}
		if _, err := inventory.LockMany(ctx, tx, VariantIDs(lines)); err != nil {
			return err
		}
		ref := inventory.Reference{Type: ReferenceOrder, ID: o.ID}
		for _, l := range lines {
			if err := inventory.Restock(ctx, tx, l.VariantID, l.Quantity, ref, &buyerID, "order refunded"); err != nil {
				return err
			}
		}
		return repo.AddSoldCounts(ctx, lines, -1)
	})
	if err != nil {
		return Order{}, err
	}
	return s.GetForBuyer(ctx, buyerID, orderID)
}
