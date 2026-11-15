package reviews

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
	CodeItemNotFound    = "ORDER_ITEM_NOT_FOUND"
	CodeNotAllowed      = "REVIEW_NOT_ALLOWED"
	CodeExists          = "REVIEW_EXISTS"
	CodeNotFound        = "REVIEW_NOT_FOUND"
	CodeProductNotFound = "PRODUCT_NOT_FOUND"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Signed-in routes: the author is always the session user.
func userID(c *gin.Context) uuid.UUID {
	p, _ := authctx.FromContext(c.Request.Context())
	return p.UserID
}

// Create handles POST /api/v1/reviews {orderItemId, rating, comment?}.
func (h *Handler) Create(c *gin.Context) {
	var body CreateReviewRequest
	if !request.BindJSON(c, &body) {
		return
	}
	in := Input{Rating: body.Rating, Comment: body.Comment}
	itemID, err := uuid.Parse(body.OrderItemID)
	if err != nil {
		// Report every field at once: the id problem plus any content problem.
		errs := validation.Errors{"orderItemId": "must be a valid id"}
		var content validation.Errors
		if _, e := in.Clean(); errors.As(e, &content) {
			for field, msg := range content {
				errs.Add(field, msg)
			}
		}
		h.fail(c, errs)
		return
	}
	r, err := h.svc.Create(c.Request.Context(), userID(c), itemID, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusCreated, NewReviewResponse(r), "Review saved")
}

// Update handles PATCH /api/v1/reviews/:id {rating?, comment?}.
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound) // a malformed id is just another unknown review
		return
	}
	var body UpdateReviewRequest
	if !request.BindJSON(c, &body) {
		return
	}
	r, err := h.svc.Update(c.Request.Context(), userID(c), id, UpdateInput(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewReviewResponse(r), "Review updated")
}

// ForProduct handles GET /api/v1/products/:id/reviews?page=&pageSize= (public).
func (h *Handler) ForProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrProductNotFound)
		return
	}
	page, pageSize, ok := h.paging(c)
	if !ok {
		return
	}
	sum, p, err := h.svc.ProductReviews(c.Request.Context(), id, page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewProductReviewsResponse(sum, p), "")
}

// Mine handles GET /api/v1/me/reviews?page=&pageSize=.
func (h *Handler) Mine(c *gin.Context) {
	page, pageSize, ok := h.paging(c)
	if !ok {
		return
	}
	p, err := h.svc.Mine(c.Request.Context(), userID(c), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewMyReviewsResponse(p), "")
}

// Pending handles GET /api/v1/me/reviews/pending?page=&pageSize=: delivered
// items of the caller that have no review yet.
func (h *Handler) Pending(c *gin.Context) {
	page, pageSize, ok := h.paging(c)
	if !ok {
		return
	}
	p, err := h.svc.Pending(c.Request.Context(), userID(c), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewPendingResponse(p), "")
}

func (h *Handler) paging(c *gin.Context) (int, int, bool) {
	errs := validation.Errors{}
	page := intQuery(c, errs, "page")
	pageSize := intQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return 0, 0, false
	}
	return page, pageSize, true
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
		notAllowed *NotAllowedError
	)
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.As(err, &notAllowed):
		response.Fail(c, http.StatusConflict, CodeNotAllowed,
			"Only items of a delivered order can be reviewed", map[string]string{"orderStatus": notAllowed.OrderStatus})
	case errors.Is(err, ErrItemNotFound):
		response.Fail(c, http.StatusNotFound, CodeItemNotFound, "Order item not found", nil)
	case errors.Is(err, ErrExists):
		response.Fail(c, http.StatusConflict, CodeExists, "This item has already been reviewed", nil)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Review not found", nil)
	case errors.Is(err, ErrProductNotFound):
		response.Fail(c, http.StatusNotFound, CodeProductNotFound, "Product not found", nil)
	default:
		response.Internal(c, err)
	}
}
