package reviews

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

// Repository wraps the sqlc queries. It takes queries.DBTX, so it works with
// the pool (reads) or a transaction (writes); it holds no business rules.
type Repository struct {
	q *queries.Queries
}

func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

func toReview(r queries.Review) Review {
	return Review{
		ID: r.ID, OrderItemID: r.OrderItemID, ProductID: r.ProductID, Rating: r.Rating,
		Comment: r.Comment, Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func image(url, alt string) *Image {
	if url == "" {
		return nil
	}
	return &Image{URL: url, Alt: alt}
}

// reviewable is what the service needs to know about an order item.
type reviewable struct {
	ProductID   uuid.UUID
	StoreID     uuid.UUID
	OrderStatus string
}

// ReviewableItem returns the item only when it belongs to one of the buyer's
// orders; anyone else's item is ErrItemNotFound.
func (r *Repository) ReviewableItem(ctx context.Context, buyerID, orderItemID uuid.UUID) (reviewable, error) {
	row, err := r.q.GetReviewableItem(ctx, queries.GetReviewableItemParams{OrderItemID: orderItemID, BuyerID: buyerID})
	if errors.Is(err, pgx.ErrNoRows) {
		return reviewable{}, ErrItemNotFound
	}
	if err != nil {
		return reviewable{}, fmt.Errorf("get reviewable item: %w", err)
	}
	return reviewable{ProductID: row.ProductID, StoreID: row.StoreID, OrderStatus: row.OrderStatus}, nil
}

// Insert returns ErrExists when the item already has a review.
func (r *Repository) Insert(ctx context.Context, userID, orderItemID uuid.UUID, item reviewable, in Input) (Review, error) {
	row, err := r.q.InsertReview(ctx, queries.InsertReviewParams{
		OrderItemID: orderItemID, ProductID: item.ProductID, StoreID: item.StoreID, UserID: userID,
		Rating: int16(in.Rating), Comment: in.Comment,
	})
	if errors.Is(err, pgx.ErrNoRows) { // ON CONFLICT DO NOTHING returned no row
		return Review{}, ErrExists
	}
	if err != nil {
		return Review{}, fmt.Errorf("insert review: %w", err)
	}
	return toReview(row), nil
}

func (r *Repository) AdjustProductRating(ctx context.Context, productID uuid.UUID, countDelta, totalDelta int) error {
	err := r.q.AdjustProductRating(ctx, queries.AdjustProductRatingParams{
		ID: productID, CountDelta: int32(countDelta), TotalDelta: int32(totalDelta),
	})
	if err != nil {
		return fmt.Errorf("adjust product rating: %w", err)
	}
	return nil
}

func (r *Repository) LockOwn(ctx context.Context, userID, id uuid.UUID) (Review, error) {
	row, err := r.q.LockOwnReview(ctx, queries.LockOwnReviewParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Review{}, ErrNotFound
	}
	if err != nil {
		return Review{}, fmt.Errorf("lock review: %w", err)
	}
	return toReview(row), nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, in Input) (Review, error) {
	row, err := r.q.UpdateReview(ctx, queries.UpdateReviewParams{ID: id, Rating: int16(in.Rating), Comment: in.Comment})
	if err != nil {
		return Review{}, fmt.Errorf("update review: %w", err)
	}
	return toReview(row), nil
}

// ProductSummary returns ErrProductNotFound for a product that is not public.
func (r *Repository) ProductSummary(ctx context.Context, productID uuid.UUID) (Summary, error) {
	agg, err := r.q.GetPublicProductRating(ctx, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, ErrProductNotFound
	}
	if err != nil {
		return Summary{}, fmt.Errorf("get product rating: %w", err)
	}
	s := Summary{Count: agg.RatingCount, Total: agg.RatingTotal}
	rows, err := r.q.ProductRatingDistribution(ctx, productID)
	if err != nil {
		return Summary{}, fmt.Errorf("rating distribution: %w", err)
	}
	for _, row := range rows {
		if row.Rating >= MinRating && row.Rating <= MaxRating {
			s.Distribution[row.Rating-1] = row.Count
		}
	}
	return s, nil
}

func (r *Repository) ListForProduct(ctx context.Context, productID uuid.UUID, limit, offset int) ([]PublicReview, int64, error) {
	rows, err := r.q.ListProductReviews(ctx, queries.ListProductReviewsParams{
		ProductID: productID, PageLimit: int32(limit), PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list product reviews: %w", err)
	}
	var total int64
	out := make([]PublicReview, len(rows))
	for i, row := range rows {
		total = row.TotalCount
		out[i] = PublicReview{
			ID: row.ID, Rating: row.Rating, Comment: row.Comment, VariantName: row.VariantName,
			ReviewerName: MaskName(row.ReviewerName), CreatedAt: row.CreatedAt,
			Edited: row.UpdatedAt.After(row.CreatedAt),
		}
	}
	return out, total, nil
}

func (r *Repository) ListMine(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MyReview, int64, error) {
	rows, err := r.q.ListMyReviews(ctx, queries.ListMyReviewsParams{
		UserID: userID, PageLimit: int32(limit), PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list my reviews: %w", err)
	}
	var total int64
	out := make([]MyReview, len(rows))
	for i, row := range rows {
		total = row.TotalCount
		out[i] = MyReview{
			Review: Review{
				ID: row.ID, OrderItemID: row.OrderItemID, ProductID: row.ProductID, Rating: row.Rating,
				Comment: row.Comment, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			},
			Item: ItemRef{
				OrderItemID: row.OrderItemID, OrderID: row.OrderID, OrderNumber: row.OrderNumber,
				ProductID: row.ProductID, ProductSlug: row.ProductSlug, StoreSlug: row.StoreSlug,
				ProductName: row.ProductName, VariantName: row.VariantName, Image: image(row.ImageUrl, row.ImageAlt),
			},
		}
	}
	return out, total, nil
}

func (r *Repository) ListPending(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]PendingItem, int64, error) {
	rows, err := r.q.ListPendingReviews(ctx, queries.ListPendingReviewsParams{
		BuyerID: buyerID, PageLimit: int32(limit), PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list pending reviews: %w", err)
	}
	var total int64
	out := make([]PendingItem, len(rows))
	for i, row := range rows {
		total = row.TotalCount
		out[i] = PendingItem{
			ItemRef: ItemRef{
				OrderItemID: row.OrderItemID, OrderID: row.OrderID, OrderNumber: row.OrderNumber,
				ProductID: row.ProductID, ProductSlug: row.ProductSlug, StoreSlug: row.StoreSlug,
				ProductName: row.ProductName, VariantName: row.VariantName, Image: image(row.ImageUrl, row.ImageAlt),
			},
			StoreName: row.StoreName, DeliveredAt: row.DeliveredAt,
		}
	}
	return out, total, nil
}
