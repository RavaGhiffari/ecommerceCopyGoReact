package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

type Repository struct {
	q *queries.Queries
}

// NewRepository works with the pool or a transaction (both are DBTX).
func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

// EnsureCart returns the user's cart id, creating the cart if needed. Inside a
// transaction the cart row stays locked until commit (see cart.sql).
func (r *Repository) EnsureCart(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	id, err := r.q.EnsureCart(ctx, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("ensure cart: %w", err)
	}
	return id, nil
}

// LockCart locks the user's existing cart. No cart means no items: ErrItemNotFound.
func (r *Repository) LockCart(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	id, err := r.q.LockCartByUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrItemNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("lock cart: %w", err)
	}
	return id, nil
}

type variantInfo struct {
	ProductStatus string
	StoreStatus   string
	StoreOwnerID  uuid.UUID
	Available     int32
}

func (r *Repository) Variant(ctx context.Context, variantID uuid.UUID) (variantInfo, error) {
	row, err := r.q.GetVariantForCart(ctx, variantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return variantInfo{}, ErrVariantNotFound
	}
	if err != nil {
		return variantInfo{}, fmt.Errorf("get variant for cart: %w", err)
	}
	return variantInfo{
		ProductStatus: row.ProductStatus, StoreStatus: row.StoreStatus,
		StoreOwnerID: row.StoreOwnerID, Available: row.AvailableQuantity,
	}, nil
}

// QuantityInCart returns 0 when the variant is not in the cart yet.
func (r *Repository) QuantityInCart(ctx context.Context, cartID, variantID uuid.UUID) (int32, error) {
	q, err := r.q.GetCartItemQuantity(ctx, queries.GetCartItemQuantityParams{CartID: cartID, VariantID: variantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get cart item quantity: %w", err)
	}
	return q, nil
}

func (r *Repository) Upsert(ctx context.Context, cartID, variantID uuid.UUID, quantity int32) error {
	if err := r.q.UpsertCartItem(ctx, queries.UpsertCartItemParams{CartID: cartID, VariantID: variantID, Quantity: quantity}); err != nil {
		return fmt.Errorf("upsert cart item: %w", err)
	}
	return nil
}

type itemRef struct {
	ID        uuid.UUID
	VariantID uuid.UUID
	Quantity  int32
}

// Item returns an item only if it belongs to cartID.
func (r *Repository) Item(ctx context.Context, cartID, itemID uuid.UUID) (itemRef, error) {
	row, err := r.q.GetCartItem(ctx, queries.GetCartItemParams{ID: itemID, CartID: cartID})
	if errors.Is(err, pgx.ErrNoRows) {
		return itemRef{}, ErrItemNotFound
	}
	if err != nil {
		return itemRef{}, fmt.Errorf("get cart item: %w", err)
	}
	return itemRef{ID: row.ID, VariantID: row.VariantID, Quantity: row.Quantity}, nil
}

func (r *Repository) SetQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int32) error {
	if err := r.q.SetCartItemQuantity(ctx, queries.SetCartItemQuantityParams{ID: itemID, CartID: cartID, Quantity: quantity}); err != nil {
		return fmt.Errorf("set cart item quantity: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, userID, itemID uuid.UUID) error {
	n, err := r.q.DeleteCartItem(ctx, queries.DeleteCartItemParams{ID: itemID, UserID: userID})
	if err != nil {
		return fmt.Errorf("delete cart item: %w", err)
	}
	if n == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *Repository) Lines(ctx context.Context, userID uuid.UUID) ([]lineRow, error) {
	rows, err := r.q.ListCartLines(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list cart lines: %w", err)
	}
	out := make([]lineRow, len(rows))
	for i, row := range rows {
		out[i] = lineRow{
			ID: row.ID, Quantity: row.Quantity, CreatedAt: row.CreatedAt,
			VariantID: row.VariantID, VariantName: row.VariantName, SKU: row.Sku, Price: row.Price,
			ProductID: row.ProductID, ProductSlug: row.ProductSlug, ProductName: row.ProductName,
			ProductStatus: row.ProductStatus,
			Store:         StoreRef{ID: row.StoreID, Slug: row.StoreSlug, Name: row.StoreName, Location: row.StoreLocation},
			StoreStatus:   row.StoreStatus, AvailableQuantity: row.AvailableQuantity,
		}
		if row.ImageUrl != "" {
			out[i].Image = &Image{URL: row.ImageUrl, Alt: row.ImageAlt}
		}
	}
	return out, nil
}
