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

const CodeNotFound = "ORDER_NOT_FOUND"

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
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Order not found", nil)
	default:
		response.Internal(c, err)
	}
}
