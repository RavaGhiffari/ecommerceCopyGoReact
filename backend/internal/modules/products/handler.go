package products

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	CodeNotFound         = "PRODUCT_NOT_FOUND"
	CodeVariantNotFound  = "VARIANT_NOT_FOUND"
	CodeStoreRequired    = "STORE_REQUIRED"
	CodeSlugTaken        = "SLUG_TAKEN"
	CodeSKUTaken         = "SKU_TAKEN"
	CodeVariantNameTaken = "VARIANT_NAME_TAKEN"
	CodeCategoryNotFound = "CATEGORY_NOT_FOUND"
	CodeStoreNotFound    = "STORE_NOT_FOUND"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ---- public -----------------------------------------------------------------

// List handles GET /api/v1/products?q=&category=&store=&sort=&page=&pageSize=
func (h *Handler) List(c *gin.Context) {
	errs := validation.Errors{}
	page := intQuery(c, errs, "page")
	pageSize := intQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return
	}

	result, err := h.svc.ListPublic(c.Request.Context(), PublicQuery{
		Q:            c.Query("q"),
		CategorySlug: c.Query("category"),
		StoreSlug:    c.Query("store"),
		Sort:         c.Query("sort"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewPageResponse(result), "")
}

// Detail handles GET /api/v1/stores/:slug/products/:productSlug
func (h *Handler) Detail(c *gin.Context) {
	d, err := h.svc.PublicDetail(c.Request.Context(), c.Param("slug"), c.Param("productSlug"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewDetailResponse(d), "")
}

// Related handles GET /api/v1/products/:id/related
func (h *Handler) Related(c *gin.Context) {
	id, ok := h.uuidParam(c, "id", CodeNotFound, "Product not found")
	if !ok {
		return
	}
	items, err := h.svc.Related(c.Request.Context(), id, 4)
	if err != nil {
		h.fail(c, err)
		return
	}
	out := make([]ProductSummaryResponse, len(items))
	for i, s := range items {
		out[i] = NewSummaryResponse(s)
	}
	response.OK(c, http.StatusOK, out, "")
}

// ---- seller (routes are behind RequireRole(seller)) -------------------------

func ownerID(c *gin.Context) uuid.UUID {
	p, _ := authctx.FromContext(c.Request.Context())
	return p.UserID
}

// ListMine handles GET /api/v1/seller/products
func (h *Handler) ListMine(c *gin.Context) {
	items, err := h.svc.ListMine(c.Request.Context(), ownerID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	out := make([]SellerProductSummaryResponse, len(items))
	for i, s := range items {
		out[i] = NewSellerSummaryResponse(s)
	}
	response.OK(c, http.StatusOK, out, "")
}

// GetMine handles GET /api/v1/seller/products/:id
func (h *Handler) GetMine(c *gin.Context) {
	id, ok := h.uuidParam(c, "id", CodeNotFound, "Product not found")
	if !ok {
		return
	}
	p, err := h.svc.GetMine(c.Request.Context(), ownerID(c), id)
	h.respondProduct(c, p, err, http.StatusOK, "")
}

// Create handles POST /api/v1/seller/products
func (h *Handler) Create(c *gin.Context) {
	var body CreateProductRequest
	if !request.BindJSON(c, &body) {
		return
	}
	variants := make([]VariantInput, len(body.Variants))
	for i, v := range body.Variants {
		variants[i] = toVariantInput(v)
	}
	p, err := h.svc.Create(c.Request.Context(), ownerID(c), CreateInput{
		Name: body.Name, Slug: body.Slug, Description: body.Description, CategoryID: body.CategoryID,
		Images: toImageInputs(body.Images), Variants: variants,
	})
	h.respondProduct(c, p, err, http.StatusCreated, "Product created")
}

// Update handles PATCH /api/v1/seller/products/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := h.uuidParam(c, "id", CodeNotFound, "Product not found")
	if !ok {
		return
	}
	var body UpdateProductRequest
	if !request.BindJSON(c, &body) {
		return
	}
	in := UpdateInput{Name: body.Name, Description: body.Description, CategoryID: body.CategoryID, Status: body.Status}
	if body.Images != nil {
		imgs := toImageInputs(*body.Images)
		in.Images = &imgs
	}
	p, err := h.svc.Update(c.Request.Context(), ownerID(c), id, in)
	h.respondProduct(c, p, err, http.StatusOK, "Product updated")
}

// Archive handles DELETE /api/v1/seller/products/:id (archives, never deletes).
func (h *Handler) Archive(c *gin.Context) {
	id, ok := h.uuidParam(c, "id", CodeNotFound, "Product not found")
	if !ok {
		return
	}
	if err := h.svc.Archive(c.Request.Context(), ownerID(c), id); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AddVariant handles POST /api/v1/seller/products/:id/variants
func (h *Handler) AddVariant(c *gin.Context) {
	id, ok := h.uuidParam(c, "id", CodeNotFound, "Product not found")
	if !ok {
		return
	}
	var body VariantRequest
	if !request.BindJSON(c, &body) {
		return
	}
	p, err := h.svc.AddVariant(c.Request.Context(), ownerID(c), id, toVariantInput(body))
	h.respondProduct(c, p, err, http.StatusCreated, "Variant added")
}

// UpdateVariant handles PATCH /api/v1/seller/products/:id/variants/:variantId
func (h *Handler) UpdateVariant(c *gin.Context) {
	id, ok := h.uuidParam(c, "id", CodeNotFound, "Product not found")
	if !ok {
		return
	}
	variantID, ok := h.uuidParam(c, "variantId", CodeVariantNotFound, "Variant not found")
	if !ok {
		return
	}
	var body UpdateVariantRequest
	if !request.BindJSON(c, &body) {
		return
	}
	p, err := h.svc.UpdateVariant(c.Request.Context(), ownerID(c), id, variantID, VariantUpdateInput(body))
	h.respondProduct(c, p, err, http.StatusOK, "Variant updated")
}

// SetStock handles PUT /api/v1/seller/products/:id/variants/:variantId/stock
func (h *Handler) SetStock(c *gin.Context) {
	id, ok := h.uuidParam(c, "id", CodeNotFound, "Product not found")
	if !ok {
		return
	}
	variantID, ok := h.uuidParam(c, "variantId", CodeVariantNotFound, "Variant not found")
	if !ok {
		return
	}
	var body SetStockRequest
	if !request.BindJSON(c, &body) {
		return
	}
	qty, err := h.svc.SetStock(c.Request.Context(), ownerID(c), id, variantID, body.Quantity, body.Reason)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, StockResponse{VariantID: variantID.String(), AvailableQuantity: qty}, "Stock updated")
}

// ---- helpers ----------------------------------------------------------------

func (h *Handler) respondProduct(c *gin.Context, p SellerProduct, err error, status int, msg string) {
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, status, NewSellerProductResponse(p), msg)
}

// uuidParam: a malformed ID cannot exist, so it is a 404 like any unknown ID.
func (h *Handler) uuidParam(c *gin.Context, name, code, msg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		response.Fail(c, http.StatusNotFound, code, msg, nil)
		return uuid.Nil, false
	}
	return id, true
}

func intQuery(c *gin.Context, errs validation.Errors, name string) int {
	raw := c.Query(name)
	if raw == "" {
		return 0 // service applies the default
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
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Product not found", nil)
	case errors.Is(err, ErrVariantNotFound):
		response.Fail(c, http.StatusNotFound, CodeVariantNotFound, "Variant not found", nil)
	case errors.Is(err, ErrNoStore):
		response.Fail(c, http.StatusConflict, CodeStoreRequired, "Create your store first", nil)
	case errors.Is(err, ErrCategoryNotFound):
		response.Fail(c, http.StatusNotFound, CodeCategoryNotFound, "Category not found", nil)
	case errors.Is(err, ErrStoreNotFound):
		response.Fail(c, http.StatusNotFound, CodeStoreNotFound, "Store not found", nil)
	case errors.Is(err, ErrSlugTaken):
		response.Fail(c, http.StatusConflict, CodeSlugTaken, "Your store already has a product with this address",
			map[string]string{"slug": "is already used in your store"})
	case errors.Is(err, ErrSKUTaken):
		response.Fail(c, http.StatusConflict, CodeSKUTaken, "This SKU is already used", map[string]string{"sku": "is already used"})
	case errors.Is(err, ErrVariantNameTaken):
		response.Fail(c, http.StatusConflict, CodeVariantNameTaken, "This product already has a variant with that name",
			map[string]string{"name": "is already used by another variant"})
	default:
		response.Internal(c, err)
	}
}
