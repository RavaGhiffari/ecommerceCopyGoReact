package checkout

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// HeaderIdempotencyKey carries a client-generated UUID, the same for every
// retry of one "place order" click.
const HeaderIdempotencyKey = "Idempotency-Key"

const (
	CodeIdempotencyKeyRequired = "IDEMPOTENCY_KEY_REQUIRED"
	CodeIdempotencyKeyReused   = "IDEMPOTENCY_KEY_REUSED"
	CodeCartEmpty              = "CART_EMPTY"
	CodeCartHasIssues          = "CART_HAS_ISSUES"
	CodeShippingNotAllowed     = "SHIPPING_METHOD_NOT_ALLOWED"
	CodeNotFound               = "CHECKOUT_NOT_FOUND"
	CodeNotPending             = "CHECKOUT_NOT_PENDING"
	CodeExpired                = "CHECKOUT_EXPIRED"
	CodePaymentNotFound        = "PAYMENT_NOT_FOUND"
	CodePaymentEventConflict   = "PAYMENT_EVENT_CONFLICT"
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

// Quote handles POST /api/v1/checkout/quote (read-only; POST because it has a body).
func (h *Handler) Quote(c *gin.Context) {
	in, ok := bindInput(c)
	if !ok {
		return
	}
	q, err := h.svc.Quote(c.Request.Context(), userID(c), in)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewQuoteResponse(q), "")
}

// Place handles POST /api/v1/checkout. 201 with the new checkout, or 200
// with the original one when the Idempotency-Key was already used for the
// same request (a retry after a timeout, a double click).
func (h *Handler) Place(c *gin.Context) {
	key, err := uuid.Parse(c.GetHeader(HeaderIdempotencyKey))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, CodeIdempotencyKeyRequired,
			"The Idempotency-Key header must be a UUID", nil)
		return
	}
	in, ok := bindInput(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	id, created, err := h.svc.Place(ctx, userID(c), key, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	co, list, err := h.svc.Get(ctx, userID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	status, msg := http.StatusCreated, "Order placed"
	if !created {
		status, msg = http.StatusOK, "Order already placed with this Idempotency-Key"
	}
	response.OK(c, status, NewCheckoutResponse(co, list), msg)
}

// Get handles GET /api/v1/checkout/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound)
		return
	}
	co, list, err := h.svc.Get(c.Request.Context(), userID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewCheckoutResponse(co, list), "")
}

// Cancel handles POST /api/v1/checkout/:id/cancel (only while unpaid).
func (h *Handler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrNotFound)
		return
	}
	ctx := c.Request.Context()
	if err := h.svc.Cancel(ctx, userID(c), id); err != nil {
		h.fail(c, err)
		return
	}
	h.respond(c, id, "Checkout cancelled")
}

// SimulatePayment handles POST /api/v1/payments/:id/simulate. It stands in
// for a payment gateway's notification (webhook) while there is no real
// gateway; the result is the checkout with its new status.
func (h *Handler) SimulatePayment(c *gin.Context) {
	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrPaymentNotFound)
		return
	}
	var body SimulatePaymentRequest
	if !request.BindJSON(c, &body) {
		return
	}
	errs := validation.Errors{}
	outcome := Outcome(body.Outcome)
	if outcome != OutcomeSuccess && outcome != OutcomeFailure {
		errs.Add("outcome", "must be success or failure")
	}
	if body.EventID == nil {
		errs.Add("eventId", "is required")
	}
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return
	}

	checkoutID, applied, err := h.svc.SimulatePayment(c.Request.Context(), userID(c), paymentID, outcome, *body.EventID)
	if err != nil {
		h.fail(c, err)
		return
	}
	msg := "Payment succeeded"
	switch {
	case !applied:
		msg = "This payment event was already processed"
	case outcome == OutcomeFailure:
		msg = "Payment failed; the order was cancelled"
	}
	h.respond(c, checkoutID, msg)
}

// respond sends the checkout (with payment and orders) as it is now.
func (h *Handler) respond(c *gin.Context, id uuid.UUID, msg string) {
	co, list, err := h.svc.Get(c.Request.Context(), userID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewCheckoutResponse(co, list), msg)
}

func bindInput(c *gin.Context) (Input, bool) {
	var body Request
	if !request.BindJSON(c, &body) {
		return Input{}, false
	}
	in, err := ParseInput(body.AddressID, body.Shipping)
	if err != nil {
		var fields validation.Errors
		errors.As(err, &fields)
		response.ValidationFailed(c, fields)
		return Input{}, false
	}
	return in, true
}

func (h *Handler) fail(c *gin.Context, err error) {
	var (
		fields     validation.Errors
		issues     *IssuesError
		notAllowed *MethodNotAllowedError
	)
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.As(err, &issues):
		list := make([]IssueResponse, len(issues.Lines))
		for i, l := range issues.Lines {
			list[i] = IssueResponse{
				ItemID: l.ItemID.String(), VariantID: l.VariantID.String(),
				ProductName: l.ProductName, VariantName: l.VariantName,
				Quantity: l.Quantity, AvailableQuantity: l.AvailableQuantity, Issue: l.Issue,
			}
		}
		response.Fail(c, http.StatusConflict, CodeCartHasIssues,
			"Some items in your cart cannot be bought right now", map[string]any{"items": list})
	case errors.As(err, &notAllowed):
		response.Fail(c, http.StatusUnprocessableEntity, CodeShippingNotAllowed,
			"A shipping method is not available for this address", notAllowed.Fields)
	case errors.Is(err, ErrCartEmpty):
		response.Fail(c, http.StatusUnprocessableEntity, CodeCartEmpty, "Your cart is empty", nil)
	case errors.Is(err, ErrAddressNotFound):
		response.Fail(c, http.StatusNotFound, addresses.CodeNotFound, "Address not found", nil)
	case errors.Is(err, ErrKeyReused):
		response.Fail(c, http.StatusConflict, CodeIdempotencyKeyReused,
			"This Idempotency-Key was already used for a different order", nil)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, CodeNotFound, "Checkout not found", nil)
	case errors.Is(err, ErrPaymentNotFound):
		response.Fail(c, http.StatusNotFound, CodePaymentNotFound, "Payment not found", nil)
	case errors.Is(err, ErrNotPending):
		response.Fail(c, http.StatusConflict, CodeNotPending, "This checkout is no longer waiting for payment", nil)
	case errors.Is(err, ErrExpired):
		response.Fail(c, http.StatusConflict, CodeExpired,
			"The payment time ran out; the order was cancelled and its stock released", nil)
	case errors.Is(err, ErrEventConflict):
		response.Fail(c, http.StatusConflict, CodePaymentEventConflict,
			"This eventId was already used for a different payment event", nil)
	default:
		response.Internal(c, err)
	}
}
