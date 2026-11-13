package cart

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
	CodeItemNotFound        = "CART_ITEM_NOT_FOUND"
	CodeVariantNotFound     = "VARIANT_NOT_FOUND"
	CodeVariantUnavailable  = "VARIANT_UNAVAILABLE"
	CodeOutOfStock          = "OUT_OF_STOCK"
	CodeQuantityExceedStock = "QUANTITY_EXCEEDS_STOCK"
	CodeOwnProduct          = "OWN_PRODUCT"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// All routes sit behind RequireAuth; the cart always belongs to the session's
// user, never to an ID from the request.
func userID(c *gin.Context) uuid.UUID {
	p, _ := authctx.FromContext(c.Request.Context())
	return p.UserID
}

// Get handles GET /api/v1/cart.
func (h *Handler) Get(c *gin.Context) {
	cart, err := h.svc.Get(c.Request.Context(), userID(c))
	h.respond(c, cart, err, http.StatusOK, "")
}

// AddItem handles POST /api/v1/cart/items.
func (h *Handler) AddItem(c *gin.Context) {
	var body AddItemRequest
	if !request.BindJSON(c, &body) {
		return
	}
	if body.VariantID == nil {
		response.ValidationFailed(c, map[string]string{"variantId": "is required"})
		return
	}
	cart, err := h.svc.AddItem(c.Request.Context(), userID(c), *body.VariantID, body.Quantity)
	h.respond(c, cart, err, http.StatusOK, "Added to cart")
}

// UpdateItem handles PATCH /api/v1/cart/items/:id.
func (h *Handler) UpdateItem(c *gin.Context) {
	id, ok := itemID(c)
	if !ok {
		return
	}
	var body UpdateItemRequest
	if !request.BindJSON(c, &body) {
		return
	}
	cart, err := h.svc.UpdateItem(c.Request.Context(), userID(c), id, body.Quantity)
	h.respond(c, cart, err, http.StatusOK, "Cart updated")
}

// RemoveItem handles DELETE /api/v1/cart/items/:id. Returns the updated cart
// (200, not 204) so the client needs no second request.
func (h *Handler) RemoveItem(c *gin.Context) {
	id, ok := itemID(c)
	if !ok {
		return
	}
	cart, err := h.svc.RemoveItem(c.Request.Context(), userID(c), id)
	h.respond(c, cart, err, http.StatusOK, "Item removed")
}

func itemID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, CodeItemNotFound, "Cart item not found", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) respond(c *gin.Context, cart Cart, err error, status int, msg string) {
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, status, NewCartResponse(cart), msg)
}

func (h *Handler) fail(c *gin.Context, err error) {
	var fields validation.Errors
	var exceeds *ErrExceedsStock
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.As(err, &exceeds):
		response.Fail(c, http.StatusConflict, CodeQuantityExceedStock, "Not enough stock for this quantity",
			map[string]any{
				"quantity":  fmt.Sprintf("only %d available", exceeds.Available),
				"available": exceeds.Available,
				"inCart":    exceeds.InCart,
			})
	case errors.Is(err, ErrItemNotFound):
		response.Fail(c, http.StatusNotFound, CodeItemNotFound, "Cart item not found", nil)
	case errors.Is(err, ErrVariantNotFound):
		response.Fail(c, http.StatusNotFound, CodeVariantNotFound, "Product variant not found", nil)
	case errors.Is(err, ErrVariantUnavailable):
		response.Fail(c, http.StatusConflict, CodeVariantUnavailable, "This product is no longer for sale", nil)
	case errors.Is(err, ErrOutOfStock):
		response.Fail(c, http.StatusConflict, CodeOutOfStock, "This product is out of stock", nil)
	case errors.Is(err, ErrOwnProduct):
		response.Fail(c, http.StatusConflict, CodeOwnProduct, "You cannot buy from your own store", nil)
	default:
		response.Internal(c, err)
	}
}
