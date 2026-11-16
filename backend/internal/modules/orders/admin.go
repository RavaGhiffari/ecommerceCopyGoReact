package orders

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// Platform orders for admins (Phase 6d): every store's orders, read-only.
// Routes are behind RequireRole(admin); there is no store or buyer filter.

// CountAll counts every order, optionally of one status (nil = all).
func (r *Repository) CountAll(ctx context.Context, status *string) (int64, error) {
	n, err := r.q.CountAllOrders(ctx, status)
	if err != nil {
		return 0, fmt.Errorf("count all orders: %w", err)
	}
	return n, nil
}

// ListAll returns one page of every store's orders, newest first, with items.
func (r *Repository) ListAll(ctx context.Context, status *string, limit, offset int32) ([]Order, error) {
	rows, err := r.q.ListAllOrders(ctx, queries.ListAllOrdersParams{Status: status, PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list all orders: %w", err)
	}
	out := make([]Order, len(rows))
	for i, row := range rows {
		out[i] = toOrder(queries.GetBuyerOrderRow(row))
	}
	return out, r.attachItems(ctx, out)
}

// GetAny returns any order with items and history.
func (r *Repository) GetAny(ctx context.Context, id uuid.UUID) (Order, error) {
	row, err := r.q.GetAnyOrder(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}
	return r.detail(ctx, toOrder(queries.GetBuyerOrderRow(row)))
}

// ListAll is the admin's platform order list; status "" = all.
func (s *Service) ListAll(ctx context.Context, status string, page, pageSize int) (Page, error) {
	errs := validation.Errors{}
	page, pageSize = paging(errs, page, pageSize)
	var filter *string
	if status != "" {
		if !slices.Contains(Statuses, status) {
			errs.Add("status", "must be one of "+strings.Join(Statuses, ", "))
		}
		filter = &status
	}
	if err := errs.Err(); err != nil {
		return Page{}, err
	}
	repo := NewRepository(s.pool)
	total, err := repo.CountAll(ctx, filter)
	if err != nil {
		return Page{}, err
	}
	items, err := repo.ListAll(ctx, filter, int32(pageSize), int32((page-1)*pageSize))
	if err != nil {
		return Page{}, err
	}
	return newPage(items, page, pageSize, total), nil
}

// GetAny returns any order (admin).
func (s *Service) GetAny(ctx context.Context, id uuid.UUID) (Order, error) {
	return NewRepository(s.pool).GetAny(ctx, id)
}

// AdminList handles GET /api/v1/admin/orders?status=&page=&pageSize=
func (h *Handler) AdminList(c *gin.Context) {
	errs := validation.Errors{}
	page := intQuery(c, errs, "page")
	pageSize := intQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return
	}
	p, err := h.svc.ListAll(c.Request.Context(), c.Query("status"), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewPageResponse(p), "")
}

// AdminGet handles GET /api/v1/admin/orders/:id
func (h *Handler) AdminGet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound)
		return
	}
	o, err := h.svc.GetAny(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewOrderResponse(o), "")
}
