package checkout

import (
	"time"

	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/orders"
)

// JSON contract, mirrored in frontend/src/types/checkout.ts (slice 5e).

// Request is the body of POST /checkout/quote and POST /checkout.
// shipping maps a store id to "regular", "express" or "same_day".
type Request struct {
	AddressID *uuid.UUID        `json:"addressId"`
	Shipping  map[string]string `json:"shipping"`
}

// SimulatePaymentRequest is the body of POST /payments/:id/simulate.
// eventId plays the gateway's notification id: send the same one again to
// simulate a repeated delivery.
type SimulatePaymentRequest struct {
	Outcome string     `json:"outcome"` // "success" or "failure"
	EventID *uuid.UUID `json:"eventId"`
}

type ImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type LineResponse struct {
	ItemID            string         `json:"itemId"`
	VariantID         string         `json:"variantId"`
	VariantName       string         `json:"variantName"`
	SKU               string         `json:"sku"`
	ProductID         string         `json:"productId"`
	ProductSlug       string         `json:"productSlug"`
	ProductName       string         `json:"productName"`
	Image             *ImageResponse `json:"image"`
	UnitPrice         string         `json:"unitPrice"`
	Quantity          int32          `json:"quantity"`
	LineTotal         string         `json:"lineTotal"`
	AvailableQuantity int32          `json:"availableQuantity"`
	Issue             *string        `json:"issue"` // null, unavailable, out_of_stock, insufficient_stock, own_product
}

type StoreRefResponse struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type ShippingOptionResponse struct {
	Method string `json:"method"`
	Fee    string `json:"fee"`
}

type GroupResponse struct {
	Store           StoreRefResponse         `json:"store"`
	Items           []LineResponse           `json:"items"`
	ShippingOptions []ShippingOptionResponse `json:"shippingOptions"`
	ShippingMethod  string                   `json:"shippingMethod"`
	ShippingFee     string                   `json:"shippingFee"`
	Subtotal        string                   `json:"subtotal"`
	Total           string                   `json:"total"`
}

type QuoteResponse struct {
	Address       addresses.AddressResponse `json:"address"`
	Stores        []GroupResponse           `json:"stores"`
	ItemsTotal    string                    `json:"itemsTotal"`
	ShippingTotal string                    `json:"shippingTotal"`
	GrandTotal    string                    `json:"grandTotal"`
	HasIssues     bool                      `json:"hasIssues"`
}

func newLineResponse(l Line) LineResponse {
	r := LineResponse{
		ItemID: l.ItemID.String(), VariantID: l.VariantID.String(), VariantName: l.VariantName, SKU: l.SKU,
		ProductID: l.ProductID.String(), ProductSlug: l.ProductSlug, ProductName: l.ProductName,
		UnitPrice: l.UnitPrice.StringFixed(2), Quantity: l.Quantity, LineTotal: l.LineTotal.StringFixed(2),
		AvailableQuantity: l.AvailableQuantity,
	}
	if l.Image != nil {
		r.Image = &ImageResponse{URL: l.Image.URL, Alt: l.Image.Alt}
	}
	if l.Issue != "" {
		issue := l.Issue
		r.Issue = &issue
	}
	return r
}

func NewQuoteResponse(q Quote) QuoteResponse {
	out := QuoteResponse{
		Address:       addresses.NewAddressResponse(q.Address),
		Stores:        make([]GroupResponse, len(q.Groups)),
		ItemsTotal:    q.ItemsTotal.StringFixed(2),
		ShippingTotal: q.ShippingTotal.StringFixed(2),
		GrandTotal:    q.GrandTotal.StringFixed(2),
		HasIssues:     q.HasIssues,
	}
	for i, g := range q.Groups {
		items := make([]LineResponse, len(g.Lines))
		for j, l := range g.Lines {
			items[j] = newLineResponse(l)
		}
		opts := make([]ShippingOptionResponse, len(g.Options))
		for j, o := range g.Options {
			opts[j] = ShippingOptionResponse{Method: string(o.Method), Fee: o.Fee.StringFixed(2)}
		}
		out.Stores[i] = GroupResponse{
			Store:           StoreRefResponse{ID: g.Store.ID.String(), Slug: g.Store.Slug, Name: g.Store.Name, Location: g.Store.Location},
			Items:           items,
			ShippingOptions: opts,
			ShippingMethod:  string(g.Method),
			ShippingFee:     g.ShippingFee.StringFixed(2),
			Subtotal:        g.Subtotal.StringFixed(2),
			Total:           g.Total.StringFixed(2),
		}
	}
	return out
}

type PaymentResponse struct {
	ID        string     `json:"id"`
	Reference string     `json:"reference"`
	Provider  string     `json:"provider"`
	Amount    string     `json:"amount"`
	Status    string     `json:"status"`
	ExpiresAt time.Time  `json:"expiresAt"`
	PaidAt    *time.Time `json:"paidAt"`
}

type CheckoutResponse struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	ItemsTotal    string                 `json:"itemsTotal"`
	ShippingTotal string                 `json:"shippingTotal"`
	GrandTotal    string                 `json:"grandTotal"`
	ExpiresAt     time.Time              `json:"expiresAt"`
	PaidAt        *time.Time             `json:"paidAt"`
	CancelledAt   *time.Time             `json:"cancelledAt"`
	CreatedAt     time.Time              `json:"createdAt"`
	Payment       PaymentResponse        `json:"payment"`
	Orders        []orders.OrderResponse `json:"orders"`
}

func NewCheckoutResponse(c Checkout, list []orders.Order) CheckoutResponse {
	out := CheckoutResponse{
		ID: c.ID.String(), Status: c.Status,
		ItemsTotal: c.ItemsTotal.StringFixed(2), ShippingTotal: c.ShippingTotal.StringFixed(2),
		GrandTotal: c.GrandTotal.StringFixed(2),
		ExpiresAt:  c.ExpiresAt.UTC(), PaidAt: utc(c.PaidAt), CancelledAt: utc(c.CancelledAt),
		CreatedAt: c.CreatedAt.UTC(),
		Payment: PaymentResponse{
			ID: c.Payment.ID.String(), Reference: c.Payment.Reference, Provider: c.Payment.Provider,
			Amount: c.Payment.Amount.StringFixed(2), Status: c.Payment.Status,
			ExpiresAt: c.Payment.ExpiresAt.UTC(), PaidAt: utc(c.Payment.PaidAt),
		},
		Orders: make([]orders.OrderResponse, len(list)),
	}
	for i, o := range list {
		out.Orders[i] = orders.NewOrderResponse(o)
	}
	return out
}

// IssueResponse is one entry of the 409 CART_HAS_ISSUES details.
type IssueResponse struct {
	ItemID            string `json:"itemId"`
	VariantID         string `json:"variantId"`
	ProductName       string `json:"productName"`
	VariantName       string `json:"variantName"`
	Quantity          int32  `json:"quantity"`
	AvailableQuantity int32  `json:"availableQuantity"`
	Issue             string `json:"issue"`
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
