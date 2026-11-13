package cart

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Get returns the caller's cart; an empty cart if they never added anything.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Cart, error) {
	rows, err := NewRepository(s.pool).Lines(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	return buildCart(rows), nil
}

// AddItem adds quantity of a variant, or increases it when already in the cart.
//
// Everything runs in one transaction that starts by locking the cart row
// (EnsureCart). Without the lock, two quick clicks could both read "2 in
// cart", both check 2+1 against the stock, and both write 3 instead of 4.
// With it, the second request waits and sees the first one's result.
func (s *Service) AddItem(ctx context.Context, userID, variantID uuid.UUID, quantity *int32) (Cart, error) {
	qty, err := checkQuantity(quantity)
	if err != nil {
		return Cart{}, err
	}

	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		cartID, err := repo.EnsureCart(ctx, userID)
		if err != nil {
			return err
		}
		v, err := repo.Variant(ctx, variantID)
		if err != nil {
			return err
		}
		if v.StoreOwnerID == userID {
			return ErrOwnProduct
		}
		if v.ProductStatus != "active" || v.StoreStatus != "active" {
			return ErrVariantUnavailable
		}
		if v.Available <= 0 {
			return ErrOutOfStock
		}

		inCart, err := repo.QuantityInCart(ctx, cartID, variantID)
		if err != nil {
			return err
		}
		total := inCart + qty
		if total > MaxQuantityPerLine {
			return validation.Errors{"quantity": fmt.Sprintf("at most %d per item (%d already in cart)", MaxQuantityPerLine, inCart)}
		}
		if total > v.Available {
			return &ErrExceedsStock{Available: v.Available, InCart: inCart}
		}
		return repo.Upsert(ctx, cartID, variantID, total)
	})
	if err != nil {
		return Cart{}, err
	}
	return s.Get(ctx, userID)
}

// UpdateItem sets a line's quantity. The item must be in the caller's cart.
func (s *Service) UpdateItem(ctx context.Context, userID, itemID uuid.UUID, quantity *int32) (Cart, error) {
	qty, err := checkQuantity(quantity)
	if err != nil {
		return Cart{}, err
	}

	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		cartID, err := repo.LockCart(ctx, userID)
		if err != nil {
			return err
		}
		item, err := repo.Item(ctx, cartID, itemID) // another user's item: not found
		if err != nil {
			return err
		}
		v, err := repo.Variant(ctx, item.VariantID)
		if err != nil {
			return err
		}
		if v.ProductStatus != "active" || v.StoreStatus != "active" {
			return ErrVariantUnavailable // the line can only be removed now
		}
		if v.Available <= 0 {
			return ErrOutOfStock
		}
		if qty > v.Available {
			return &ErrExceedsStock{Available: v.Available}
		}
		return repo.SetQuantity(ctx, cartID, itemID, qty)
	})
	if err != nil {
		return Cart{}, err
	}
	return s.Get(ctx, userID)
}

// RemoveItem deletes a line from the caller's cart.
func (s *Service) RemoveItem(ctx context.Context, userID, itemID uuid.UUID) (Cart, error) {
	if err := NewRepository(s.pool).Delete(ctx, userID, itemID); err != nil {
		return Cart{}, err
	}
	return s.Get(ctx, userID)
}

func checkQuantity(q *int32) (int32, error) {
	switch {
	case q == nil:
		return 0, validation.Errors{"quantity": "is required"}
	case *q < 1 || *q > MaxQuantityPerLine:
		return 0, validation.Errors{"quantity": fmt.Sprintf("must be between 1 and %d", MaxQuantityPerLine)}
	}
	return *q, nil
}
