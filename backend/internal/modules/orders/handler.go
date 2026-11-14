package orders

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	CodeNotFound          = "ORDER_NOT_FOUND"
	CodeInvalidTransition = "INVALID_STATUS_TRANSITION"
	CodeStoreRequired     = "STORE_REQUIRED" // same code as the products module
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Behind RequireAuth: the buyer is always the session user.
func userID(c *gin.Context) uuid.UUID {
	p, _ := authctx.FromContext(c.Request.Context())
	return p.UserID
}

// List handles GET /api/v1/orders?page=&pageSize=
func (h *Handler) List(c *gin.Context) {
	errs := validation.Errors{}
	page := intQuery(c, errs, "page")
	pageSize := intQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return
	}
	p, err := h.svc.ListForBuyer(c.Request.Context(), userID(c), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewPageResponse(p), "")
}

// Get handles GET /api/v1/orders/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound) // a malformed id is just another unknown order
		return
	}
	o, err := h.svc.GetForBuyer(c.Request.Context(), userID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewOrderResponse(o), "")
}

// Cancel handles POST /api/v1/orders/:id/cancel (a paid order -> refunded).
func (h *Handler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound)
		return
	}
	o, err := h.svc.CancelForBuyer(c.Request.Context(), userID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewOrderResponse(o), "Order cancelled; refund simulated")
}

func intQuery(c *gin.Context, errs validation.Errors, name string) int {
	raw := c.Query(name)
	if raw == "" {
		return 0 // the service applies the default
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		errs.Add(name, "must be a whole number")
	}
	return n
}

func (h *Handler) fail(c *gin.Context, err error) {
	var (
		fields     validation.Errors
		transition *TransitionError
		unpaid     *UnpaidError
	)
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.As(err, &transition):
		response.Fail(c, http.StatusConflict, CodeInvalidTransition,
			"This order can no longer be changed that way",
			map[string]any{"from": transition.From, "to": transition.To})
	case errors.As(err, &unpaid):
		response.Fail(c, http.StatusConflict, CodeInvalidTransition,
			"An unpaid order is cancelled together with its checkout",
			map[string]any{"from": StatusPendingPayment, "checkoutId": unpaid.CheckoutID})
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Order not found", nil)
	case errors.Is(err, ErrNoStore):
		response.Fail(c, http.StatusConflict, CodeStoreRequired, "Create your store first", nil)
	default:
		response.Internal(c, err)
	}
}
