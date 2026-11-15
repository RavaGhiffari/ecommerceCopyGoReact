package wishlist

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

const CodeProductNotFound = "PRODUCT_NOT_FOUND"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// All routes sit behind RequireAuth: the owner is always the session user,
// never an id from the request.
func userID(c *gin.Context) uuid.UUID {
	p, _ := authctx.FromContext(c.Request.Context())
	return p.UserID
}

// Add handles PUT /api/v1/me/wishlist/:productId. PUT because it is
// idempotent: the same request twice gives the same result (204 both times).
func (h *Handler) Add(c *gin.Context) {
	productID, ok := productParam(c)
	if !ok {
		return
	}
	if err := h.svc.Add(c.Request.Context(), userID(c), productID); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Remove handles DELETE /api/v1/me/wishlist/:productId (204, also when the
// product was not in the wishlist).
func (h *Handler) Remove(c *gin.Context) {
	productID, ok := productParam(c)
	if !ok {
		return
	}
	if err := h.svc.Remove(c.Request.Context(), userID(c), productID); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// List handles GET /api/v1/me/wishlist?page=&pageSize=.
func (h *Handler) List(c *gin.Context) {
	errs := validation.Errors{}
	page := intQuery(c, errs, "page")
	pageSize := intQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return
	}
	p, err := h.svc.List(c.Request.Context(), userID(c), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewPageResponse(p), "")
}

// IDs handles GET /api/v1/me/wishlist/ids.
func (h *Handler) IDs(c *gin.Context) {
	ids, err := h.svc.ProductIDs(c.Request.Context(), userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	out := IDsResponse{ProductIDs: make([]string, len(ids))}
	for i, id := range ids {
		out.ProductIDs[i] = id.String()
	}
	response.OK(c, http.StatusOK, out, "")
}

// A malformed id cannot be a product, so it gets the same 404.
func productParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, CodeProductNotFound, "Product not found", nil)
		return uuid.Nil, false
	}
	return id, true
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
	case errors.Is(err, ErrProductNotFound):
		response.Fail(c, http.StatusNotFound, CodeProductNotFound, "Product not found", nil)
	default:
		response.Internal(c, err)
	}
}
