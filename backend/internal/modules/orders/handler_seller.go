package orders

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// Seller routes sit behind RequireRole(seller); the store is resolved from
// the session user in the service, never taken from the request.

// UpdateStatusRequest is the body of PATCH /seller/orders/:id/status.
type UpdateStatusRequest struct {
	Status string  `json:"status"` // processing, shipped or delivered
	Note   *string `json:"note"`   // optional; a default note is written otherwise
}

// SellerList handles GET /api/v1/seller/orders?status=&page=&pageSize=
func (h *Handler) SellerList(c *gin.Context) {
	errs := validation.Errors{}
	page := intQuery(c, errs, "page")
	pageSize := intQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return
	}
	p, err := h.svc.ListForStore(c.Request.Context(), userID(c), c.Query("status"), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewPageResponse(p), "")
}

// SellerGet handles GET /api/v1/seller/orders/:id
func (h *Handler) SellerGet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound)
		return
	}
	o, err := h.svc.GetForStore(c.Request.Context(), userID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewOrderResponse(o), "")
}

// SellerUpdateStatus handles PATCH /api/v1/seller/orders/:id/status
func (h *Handler) SellerUpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound)
		return
	}
	var body UpdateStatusRequest
	if !request.BindJSON(c, &body) {
		return
	}
	note := ""
	if body.Note != nil {
		note = *body.Note
	}
	o, err := h.svc.UpdateStatusForStore(c.Request.Context(), userID(c), id, body.Status, note)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewOrderResponse(o), "Order status updated")
}

// SellerStats handles GET /api/v1/seller/stats (the seller dashboard, 6c).
func (h *Handler) SellerStats(c *gin.Context) {
	st, err := h.svc.StatsForStore(c.Request.Context(), userID(c), time.Now())
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewSellerStatsResponse(st), "")
}
