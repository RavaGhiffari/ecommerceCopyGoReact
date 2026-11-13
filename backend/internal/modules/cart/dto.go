package cart

import "github.com/google/uuid"

// JSON contract, mirrored in frontend/src/types/cart.ts — keep them in sync.
// Money is a decimal string with two decimals, like everywhere else.

type ImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type LineResponse struct {
	ID                string         `json:"id"`
	Quantity          int32          `json:"quantity"`
	VariantID         string         `json:"variantId"`
	VariantName       string         `json:"variantName"`
	SKU               string         `json:"sku"`
	ProductID         string         `json:"productId"`
	ProductSlug       string         `json:"productSlug"`
	ProductName       string         `json:"productName"`
	Image             *ImageResponse `json:"image"` // null when the product has no image
	UnitPrice         string         `json:"unitPrice"`
	LineTotal         string         `json:"lineTotal"`
	AvailableQuantity int32          `json:"availableQuantity"`
	Issue             *string        `json:"issue"` // null, "unavailable", "out_of_stock", "insufficient_stock"
}

type StoreRefResponse struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type StoreGroupResponse struct {
	Store    StoreRefResponse `json:"store"`
	Items    []LineResponse   `json:"items"`
	Subtotal string           `json:"subtotal"`
}

type CartResponse struct {
	Stores    []StoreGroupResponse `json:"stores"`
	ItemCount int32                `json:"itemCount"`
	Subtotal  string               `json:"subtotal"`
	HasIssues bool                 `json:"hasIssues"`
}

func NewCartResponse(c Cart) CartResponse {
	out := CartResponse{
		Stores:    make([]StoreGroupResponse, len(c.Groups)),
		ItemCount: c.ItemCount,
		Subtotal:  c.Subtotal.StringFixed(2),
		HasIssues: c.HasIssues,
	}
	for i, g := range c.Groups {
		items := make([]LineResponse, len(g.Lines))
		for j, l := range g.Lines {
			items[j] = LineResponse{
				ID: l.ID.String(), Quantity: l.Quantity, VariantID: l.VariantID.String(), VariantName: l.VariantName,
				SKU: l.SKU, ProductID: l.ProductID.String(), ProductSlug: l.ProductSlug, ProductName: l.ProductName,
				UnitPrice: l.UnitPrice.StringFixed(2), LineTotal: l.LineTotal.StringFixed(2),
				AvailableQuantity: l.AvailableQuantity,
			}
			if l.Image != nil {
				items[j].Image = &ImageResponse{URL: l.Image.URL, Alt: l.Image.Alt}
			}
			if l.Issue != "" {
				issue := l.Issue
				items[j].Issue = &issue
			}
		}
		out.Stores[i] = StoreGroupResponse{
			Store:    StoreRefResponse{ID: g.Store.ID.String(), Slug: g.Store.Slug, Name: g.Store.Name, Location: g.Store.Location},
			Items:    items,
			Subtotal: g.Subtotal.StringFixed(2),
		}
	}
	return out
}

// AddItemRequest is POST /api/v1/cart/items.
type AddItemRequest struct {
	VariantID *uuid.UUID `json:"variantId"`
	Quantity  *int32     `json:"quantity"`
}

// UpdateItemRequest is PATCH /api/v1/cart/items/:id.
type UpdateItemRequest struct {
	Quantity *int32 `json:"quantity"`
}
