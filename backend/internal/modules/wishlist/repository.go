package wishlist

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

// Repository wraps the sqlc queries; it holds no business rules.
type Repository struct {
	q *queries.Queries
}

func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

func (r *Repository) IsPublicProduct(ctx context.Context, productID uuid.UUID) (bool, error) {
	ok, err := r.q.IsPublicProduct(ctx, productID)
	if err != nil {
		return false, fmt.Errorf("check product: %w", err)
	}
	return ok, nil
}

func (r *Repository) Add(ctx context.Context, userID, productID uuid.UUID) error {
	if err := r.q.AddWishlistItem(ctx, queries.AddWishlistItemParams{UserID: userID, ProductID: productID}); err != nil {
		return fmt.Errorf("add wishlist item: %w", err)
	}
	return nil
}

func (r *Repository) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	if err := r.q.RemoveWishlistItem(ctx, queries.RemoveWishlistItemParams{UserID: userID, ProductID: productID}); err != nil {
		return fmt.Errorf("remove wishlist item: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Item, int64, error) {
	rows, err := r.q.ListWishlistItems(ctx, queries.ListWishlistItemsParams{
		UserID: userID, PageLimit: int32(limit), PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list wishlist: %w", err)
	}
	var total int64
	items := make([]Item, len(rows))
	for i, row := range rows {
		total = row.TotalCount // the same on every row (window function)
		items[i] = Item{
			ProductID: row.ProductID, Slug: row.Slug, Name: row.Name, LowestPrice: row.MinPrice,
			StoreSlug: row.StoreSlug, StoreName: row.StoreName, AddedAt: row.AddedAt,
		}
		if row.ImageUrl != "" {
			items[i].Image = &Image{URL: row.ImageUrl, Alt: row.ImageAlt}
		}
	}
	return items, total, nil
}

func (r *Repository) ProductIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.q.ListWishlistProductIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list wishlist ids: %w", err)
	}
	return ids, nil
}
