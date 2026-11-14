package addresses

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	CodeNotFound     = "ADDRESS_NOT_FOUND"
	CodeLimitReached = "ADDRESS_LIMIT_REACHED"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// All routes sit behind RequireAuth. The owner is always the session's user,
// never an id from the request.
func userID(c *gin.Context) uuid.UUID {
	p, _ := authctx.FromContext(c.Request.Context())
	return p.UserID
}

// List handles GET /api/v1/me/addresses.
func (h *Handler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewAddressListResponse(list), "")
}

// Create handles POST /api/v1/me/addresses.
func (h *Handler) Create(c *gin.Context) {
	var body CreateAddressRequest
	if !request.BindJSON(c, &body) {
		return
	}
	a, err := h.svc.Create(c.Request.Context(), userID(c), body.fields(), body.IsDefault)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusCreated, NewAddressResponse(a), "Address saved")
}

// Update handles PATCH /api/v1/me/addresses/:id.
func (h *Handler) Update(c *gin.Context) {
	id, ok := addressID(c)
	if !ok {
		return
	}
	var body UpdateAddressRequest
	if !request.BindJSON(c, &body) {
		return
	}
	a, err := h.svc.Update(c.Request.Context(), userID(c), id, UpdateInput(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewAddressResponse(a), "Address updated")
}

// Delete handles DELETE /api/v1/me/addresses/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, ok := addressID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID(c), id); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// SetDefault handles POST /api/v1/me/addresses/:id/default.
func (h *Handler) SetDefault(c *gin.Context) {
	id, ok := addressID(c)
	if !ok {
		return
	}
	a, err := h.svc.SetDefault(c.Request.Context(), userID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewAddressResponse(a), "Default address changed")
}

// A malformed id cannot exist, so it gets the same 404 as a missing one.
func addressID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Address not found", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) fail(c *gin.Context, err error) {
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Address not found", nil)
	case errors.Is(err, ErrLimitReached):
		response.Fail(c, http.StatusConflict, CodeLimitReached,
			fmt.Sprintf("You can save at most %d addresses", MaxPerUser), nil)
	default:
		response.Internal(c, err)
	}
}
