package reviews

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/audit"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// Admin moderation (Phase 6d, PLAN.md Q8). It lives here because hiding a
// review must move the product's rating counters, and those are owned by
// this package (AdjustProductRating).

var reviewStatuses = []string{StatusVisible, StatusHidden}

// AdminReview is one row of the moderation list. Unlike the public list it
// names the reviewer in full: admins need to know who wrote what.
type AdminReview struct {
	ID            uuid.UUID
	Rating        int16
	Comment       string
	Status        string
	ProductID     uuid.UUID
	ProductName   string
	StoreName     string
	ReviewerName  string
	ReviewerEmail string
	Edited        bool
	CreatedAt     time.Time
}

// AdminList returns every review, newest first; status "" = both.
func (s *Service) AdminList(ctx context.Context, status string, page, pageSize int) (Page[AdminReview], error) {
	errs := validation.Errors{}
	page, pageSize, err := paging(page, pageSize)
	if err != nil {
		_ = errors.As(err, &errs) // paging reports validation.Errors
	}
	var filter *string
	if status != "" {
		if !slices.Contains(reviewStatuses, status) {
			errs.Add("status", "must be one of visible, hidden")
		}
		filter = &status
	}
	if err := errs.Err(); err != nil {
		return Page[AdminReview]{}, err
	}

	db := queries.New(s.pool)
	total, err := db.CountAdminReviews(ctx, filter)
	if err != nil {
		return Page[AdminReview]{}, fmt.Errorf("count reviews: %w", err)
	}
	rows, err := db.ListAdminReviews(ctx, queries.ListAdminReviewsParams{
		Status: filter, PageLimit: int32(pageSize), PageOffset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return Page[AdminReview]{}, fmt.Errorf("list reviews: %w", err)
	}
	items := make([]AdminReview, len(rows))
	for i, r := range rows {
		items[i] = AdminReview{
			ID: r.ID, Rating: r.Rating, Comment: r.Comment, Status: r.Status, ProductID: r.ProductID,
			ProductName: r.ProductName, StoreName: r.StoreName, ReviewerName: r.ReviewerName,
			ReviewerEmail: r.ReviewerEmail, Edited: r.UpdatedAt.After(r.CreatedAt), CreatedAt: r.CreatedAt,
		}
	}
	return Page[AdminReview]{Items: items, Page: page, PageSize: pageSize, Total: total, TotalPages: pageCount(total, pageSize)}, nil
}

// SetStatus hides or unhides a review. One transaction:
//
//  1. Lock the review row (unknown → 404). An author edit locks the same row,
//     so the two never apply their counter deltas against a stale state.
//  2. Same status → no change, no audit row.
//  3. Move the review out of (hide) or back into (unhide) the product's
//     rating counters with the atomic AdjustProductRating.
//  4. Set the status (updated_at is untouched since migration 000015, so the
//     public list does not call it "edited").
//  5. Audit row, same transaction.
func (s *Service) SetStatus(ctx context.Context, actorID, id uuid.UUID, status string) (Review, error) {
	errs := validation.Errors{}
	if status == "" {
		errs.Add("status", "is required")
	} else if !slices.Contains(reviewStatuses, status) {
		errs.Add("status", "must be one of visible, hidden")
	}
	if err := errs.Err(); err != nil {
		return Review{}, err
	}

	var out Review
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		cur, err := q.LockReview(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock review: %w", err)
		}
		out = toReview(cur)
		if cur.Status == status {
			return nil
		}

		sign, action := 1, audit.ReviewUnhide
		if status == StatusHidden {
			sign, action = -1, audit.ReviewHide
		}
		if err := NewRepository(tx).AdjustProductRating(ctx, cur.ProductID, sign, sign*int(cur.Rating)); err != nil {
			return err
		}
		updated, err := q.SetReviewStatus(ctx, queries.SetReviewStatusParams{ID: id, Status: status})
		if err != nil {
			return fmt.Errorf("set review status: %w", err)
		}
		out = toReview(updated)
		return audit.Write(ctx, tx, audit.Entry{
			ActorID: actorID, Action: action, TargetType: audit.TargetReview, TargetID: id,
			Details: map[string]any{"productId": cur.ProductID, "rating": cur.Rating},
		})
	})
	if err != nil {
		return Review{}, err
	}
	return out, nil
}

// ---- HTTP (admin routes, behind RequireRole(admin)) ------------------------------

type AdminReviewResponse struct {
	ID      string `json:"id"`
	Rating  int16  `json:"rating"`
	Comment string `json:"comment"`
	Status  string `json:"status"`
	Edited  bool   `json:"edited"`
	Product struct {
		ID   string `json:"id"`
		Name string `json:"name"` // order item snapshot
	} `json:"product"`
	StoreName string `json:"storeName"`
	Reviewer  struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"reviewer"`
	CreatedAt time.Time `json:"createdAt"`
}

func newAdminReviewResponse(r AdminReview) AdminReviewResponse {
	out := AdminReviewResponse{ID: r.ID.String(), Rating: r.Rating, Comment: r.Comment, Status: r.Status,
		Edited: r.Edited, StoreName: r.StoreName, CreatedAt: r.CreatedAt.UTC()}
	out.Product.ID, out.Product.Name = r.ProductID.String(), r.ProductName
	out.Reviewer.Name, out.Reviewer.Email = r.ReviewerName, r.ReviewerEmail
	return out
}

// AdminList handles GET /api/v1/admin/reviews?status=&page=&pageSize=
func (h *Handler) AdminList(c *gin.Context) {
	errs := validation.Errors{}
	page := intQuery(c, errs, "page")
	pageSize := intQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return
	}
	p, err := h.svc.AdminList(c.Request.Context(), c.Query("status"), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	items := make([]AdminReviewResponse, len(p.Items))
	for i, r := range p.Items {
		items[i] = newAdminReviewResponse(r)
	}
	response.OK(c, http.StatusOK, PageResponse[AdminReviewResponse]{
		Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages,
	}, "")
}

// AdminSetStatus handles PATCH /api/v1/admin/reviews/:id/status {status}
func (h *Handler) AdminSetStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound)
		return
	}
	var body struct {
		Status string `json:"status"` // visible or hidden
	}
	if !request.BindJSON(c, &body) {
		return
	}
	p, _ := authctx.FromContext(c.Request.Context())
	r, err := h.svc.SetStatus(c.Request.Context(), p.UserID, id, body.Status)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewReviewResponse(r), "Review status updated")
}
