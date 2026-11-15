package reviews

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// orderDelivered mirrors orders.StatusDelivered. Not imported, so reviews
// does not depend on the orders package for one constant.
const orderDelivered = "delivered"

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Create reviews one order item of the caller. One transaction:
//
//  1. Load the item, filtered by buyer_id = caller (another buyer's -> 404).
//  2. Its order must be delivered (409 otherwise). "delivered" is final, so
//     no lock is needed: the status cannot change after this check.
//  3. INSERT … ON CONFLICT (order_item_id) DO NOTHING. A second review of the
//     same item (also a concurrent one, which waits on the unique index) gets
//     no row back -> 409 REVIEW_EXISTS.
//  4. Add the stars to the product's counters with an atomic UPDATE.
//
// A failure in 4 rolls back 3, so the counters always match the reviews.
func (s *Service) Create(ctx context.Context, userID, orderItemID uuid.UUID, in Input) (Review, error) {
	in, err := in.Clean()
	if err != nil {
		return Review{}, err
	}
	var created Review
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		item, err := repo.ReviewableItem(ctx, userID, orderItemID)
		if err != nil {
			return err
		}
		if item.OrderStatus != orderDelivered {
			return &NotAllowedError{OrderStatus: item.OrderStatus}
		}
		if created, err = repo.Insert(ctx, userID, orderItemID, item, in); err != nil {
			return err
		}
		return repo.AdjustProductRating(ctx, item.ProductID, 1, in.Rating)
	})
	return created, err
}

// UpdateInput: nil leaves a field unchanged.
type UpdateInput struct {
	Rating  *int
	Comment *string
}

// Update edits the caller's review. The review row is locked, so two edits
// serialise and each one's delta (new - old stars) is computed from what the
// other left behind. A hidden review is outside the aggregate, so editing it
// does not touch the counters.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (Review, error) {
	if in.Rating == nil && in.Comment == nil {
		errs := validation.Errors{}
		errs.Add("rating", "rating or comment is required")
		return Review{}, errs
	}
	var updated Review
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		cur, err := repo.LockOwn(ctx, userID, id)
		if err != nil {
			return err
		}
		next := Input{Rating: int(cur.Rating), Comment: cur.Comment}
		if in.Rating != nil {
			next.Rating = *in.Rating
		}
		if in.Comment != nil {
			next.Comment = *in.Comment
		}
		if next, err = next.Clean(); err != nil {
			return err
		}
		if updated, err = repo.Update(ctx, id, next); err != nil {
			return err
		}
		if delta := next.Rating - int(cur.Rating); delta != 0 && cur.Status == StatusVisible {
			return repo.AdjustProductRating(ctx, cur.ProductID, 0, delta)
		}
		return nil
	})
	return updated, err
}

// ProductReviews is the public review block of a product page.
func (s *Service) ProductReviews(ctx context.Context, productID uuid.UUID, page, pageSize int) (Summary, Page[PublicReview], error) {
	page, pageSize, err := paging(page, pageSize)
	if err != nil {
		return Summary{}, Page[PublicReview]{}, err
	}
	repo := NewRepository(s.pool)
	sum, err := repo.ProductSummary(ctx, productID)
	if err != nil {
		return Summary{}, Page[PublicReview]{}, err
	}
	items, total, err := repo.ListForProduct(ctx, productID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Summary{}, Page[PublicReview]{}, err
	}
	if len(items) == 0 && page > 1 {
		total = int64(sum.Count) // past the last page; the counter is the total
	}
	return sum, Page[PublicReview]{Items: items, Page: page, PageSize: pageSize, Total: total, TotalPages: pageCount(total, pageSize)}, nil
}

func (s *Service) Mine(ctx context.Context, userID uuid.UUID, page, pageSize int) (Page[MyReview], error) {
	page, pageSize, err := paging(page, pageSize)
	if err != nil {
		return Page[MyReview]{}, err
	}
	repo := NewRepository(s.pool)
	items, total, err := repo.ListMine(ctx, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[MyReview]{}, err
	}
	if len(items) == 0 && page > 1 {
		if _, total, err = repo.ListMine(ctx, userID, 1, 0); err != nil {
			return Page[MyReview]{}, err
		}
	}
	return Page[MyReview]{Items: items, Page: page, PageSize: pageSize, Total: total, TotalPages: pageCount(total, pageSize)}, nil
}

func (s *Service) Pending(ctx context.Context, userID uuid.UUID, page, pageSize int) (Page[PendingItem], error) {
	page, pageSize, err := paging(page, pageSize)
	if err != nil {
		return Page[PendingItem]{}, err
	}
	repo := NewRepository(s.pool)
	items, total, err := repo.ListPending(ctx, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[PendingItem]{}, err
	}
	if len(items) == 0 && page > 1 {
		if _, total, err = repo.ListPending(ctx, userID, 1, 0); err != nil {
			return Page[PendingItem]{}, err
		}
	}
	return Page[PendingItem]{Items: items, Page: page, PageSize: pageSize, Total: total, TotalPages: pageCount(total, pageSize)}, nil
}

// paging applies defaults (0 = default) and limits.
func paging(page, pageSize int) (int, int, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	errs := validation.Errors{}
	if page < 1 || page > 6_000 {
		errs.Add("page", "must be between 1 and 10000")
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		errs.Add("pageSize", "must be between 1 and 50")
	}
	return page, pageSize, errs.Err()
}

// NotAllowedError says why an item cannot be reviewed (its order status).
type NotAllowedError struct {
	OrderStatus string
}

func (e *NotAllowedError) Error() string { return "order is " + e.OrderStatus + ", not delivered" }
func (e *NotAllowedError) Unwrap() error { return ErrNotAllowed }
