package stores

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	CodeNotFound    = "STORE_NOT_FOUND"
	CodeStoreExists = "STORE_EXISTS"
	CodeSlugTaken   = "SLUG_TAKEN"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Open handles POST /api/v1/seller/store (any signed-in user).
func (h *Handler) Open(c *gin.Context) {
	p, _ := authctx.FromContext(c.Request.Context())
	var body OpenStoreRequest
	if !request.BindJSON(c, &body) {
		return
	}
	st, err := h.svc.Open(c.Request.Context(), p.UserID, OpenInput(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusCreated, NewSellerStoreResponse(st), "Store created")
}

// Mine handles GET /api/v1/seller/store.
func (h *Handler) Mine(c *gin.Context) {
	p, _ := authctx.FromContext(c.Request.Context())
	st, err := h.svc.Mine(c.Request.Context(), p.UserID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewSellerStoreResponse(st), "")
}

// UpdateMine handles PATCH /api/v1/seller/store.
func (h *Handler) UpdateMine(c *gin.Context) {
	p, _ := authctx.FromContext(c.Request.Context())
	var body UpdateStoreRequest
	if !request.BindJSON(c, &body) {
		return
	}
	st, err := h.svc.UpdateMine(c.Request.Context(), p.UserID, UpdateInput(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewSellerStoreResponse(st), "Store updated")
}

// Public handles GET /api/v1/stores/:slug.
func (h *Handler) Public(c *gin.Context) {
	st, err := h.svc.Public(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewPublicStoreResponse(st), "")
}

func (h *Handler) fail(c *gin.Context, err error) {
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Store not found", nil)
	case errors.Is(err, ErrAlreadyHasStore):
		response.Fail(c, http.StatusConflict, CodeStoreExists, "You already have a store", nil)
	case errors.Is(err, ErrSlugTaken):
		response.Fail(c, http.StatusConflict, CodeSlugTaken, "This store address is already used", map[string]string{"slug": "is already used"})
	default:
		response.Internal(c, err)
	}
}
