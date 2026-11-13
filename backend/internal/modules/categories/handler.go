package categories

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	CodeNotFound  = "CATEGORY_NOT_FOUND"
	CodeSlugTaken = "SLUG_TAKEN"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/categories (public).
func (h *Handler) List(c *gin.Context) {
	cats, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Internal(c, err)
		return
	}
	out := make([]CategoryResponse, len(cats))
	for i, cat := range cats {
		out[i] = NewCategoryResponse(cat)
	}
	response.OK(c, http.StatusOK, out, "")
}

// Create handles POST /api/v1/admin/categories (admin only, see router).
func (h *Handler) Create(c *gin.Context) {
	var body CategoryRequest
	if !request.BindJSON(c, &body) {
		return
	}
	cat, err := h.svc.Create(c.Request.Context(), Input(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusCreated, NewCategoryResponse(cat), "Category created")
}

// Replace handles PUT /api/v1/admin/categories/:id (admin only).
func (h *Handler) Replace(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Category not found", nil)
		return
	}
	var body CategoryRequest
	if !request.BindJSON(c, &body) {
		return
	}
	cat, err := h.svc.Replace(c.Request.Context(), id, Input(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewCategoryResponse(cat), "Category updated")
}

func (h *Handler) fail(c *gin.Context, err error) {
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Category not found", nil)
	case errors.Is(err, ErrSlugTaken):
		response.Fail(c, http.StatusConflict, CodeSlugTaken, "This slug is already used", map[string]string{"slug": "is already used"})
	case errors.Is(err, ErrTooDeep):
		response.ValidationFailed(c, map[string]string{"parentId": "only two category levels are allowed"})
	default:
		response.Internal(c, err)
	}
}
