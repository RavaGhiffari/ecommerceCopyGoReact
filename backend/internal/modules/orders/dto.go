package orders

import "time"

// JSON contract, mirrored in frontend/src/types/orders.ts (slice 5e).
// Money is a decimal string with two decimals; times are RFC 3339 UTC.

type ImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ItemResponse struct {
	ID          string         `json:"id"`
	VariantID   string         `json:"variantId"`
	ProductID   string         `json:"productId"`
	ProductSlug string         `json:"productSlug"`
	ProductName string         `json:"productName"`
	VariantName string         `json:"variantName"`
	SKU         string         `json:"sku"`
	UnitPrice   string         `json:"unitPrice"`
	Quantity    int32          `json:"quantity"`
	LineTotal   string         `json:"lineTotal"`
	Image       *ImageResponse `json:"image"`
	// The buyer's review of this item (null if none). Lets the order page
	// show "Beri ulasan" / "Ubah ulasan" without one request per item.
	Review *ItemReviewResponse `json:"review"`
}

type ItemReviewResponse struct {
	ID      string `json:"id"`
	Rating  int16  `json:"rating"`
	Comment string `json:"comment"` // so the edit dialog can start from it
}

type StoreRefResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type ShippingAddressResponse struct {
	RecipientName string `json:"recipientName"`
	Phone         string `json:"phone"`
	AddressLine   string `json:"addressLine"`
	City          string `json:"city"`
	Province      string `json:"province"`
	PostalCode    string `json:"postalCode"`
}

type HistoryResponse struct {
	FromStatus *string   `json:"fromStatus"`
	ToStatus   string    `json:"toStatus"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
}

type OrderResponse struct {
	ID              string                  `json:"id"`
	CheckoutID      string                  `json:"checkoutId"`
	OrderNumber     string                  `json:"orderNumber"`
	Status          string                  `json:"status"`
	Store           StoreRefResponse        `json:"store"`
	ShippingMethod  string                  `json:"shippingMethod"`
	ShippingFee     string                  `json:"shippingFee"`
	ItemsSubtotal   string                  `json:"itemsSubtotal"`
	Total           string                  `json:"total"`
	ShippingAddress ShippingAddressResponse `json:"shippingAddress"`
	Items           []ItemResponse          `json:"items"`
	// Only in GET /orders/:id.
	History   []HistoryResponse `json:"history,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func NewOrderResponse(o Order) OrderResponse {
	items := make([]ItemResponse, len(o.Items))
	for i, it := range o.Items {
		items[i] = ItemResponse{
			ID: it.ID.String(), VariantID: it.VariantID.String(), ProductID: it.ProductID.String(),
			ProductSlug: it.ProductSlug, ProductName: it.ProductName, VariantName: it.VariantName, SKU: it.SKU,
			UnitPrice: it.UnitPrice.StringFixed(2), Quantity: it.Quantity, LineTotal: it.LineTotal.StringFixed(2),
		}
		if it.Image != nil {
			items[i].Image = &ImageResponse{URL: it.Image.URL, Alt: it.Image.Alt}
		}
		if it.Review != nil {
			items[i].Review = &ItemReviewResponse{ID: it.Review.ID.String(), Rating: it.Review.Rating, Comment: it.Review.Comment}
		}
	}
	var history []HistoryResponse
	for _, h := range o.History {
		history = append(history, HistoryResponse{FromStatus: h.FromStatus, ToStatus: h.ToStatus, Note: h.Note, CreatedAt: h.CreatedAt.UTC()})
	}
	return OrderResponse{
		ID: o.ID.String(), CheckoutID: o.CheckoutID.String(), OrderNumber: o.OrderNumber, Status: o.Status,
		Store:          StoreRefResponse{ID: o.StoreID.String(), Slug: o.StoreSlug, Name: o.StoreName},
		ShippingMethod: o.ShippingMethod, ShippingFee: o.ShippingFee.StringFixed(2),
		ItemsSubtotal: o.ItemsSubtotal.StringFixed(2), Total: o.Total.StringFixed(2),
		ShippingAddress: ShippingAddressResponse{
			RecipientName: o.ShipTo.RecipientName, Phone: o.ShipTo.Phone, AddressLine: o.ShipTo.AddressLine,
			City: o.ShipTo.City, Province: o.ShipTo.Province, PostalCode: o.ShipTo.PostalCode,
		},
		Items: items, History: history,
		CreatedAt: o.CreatedAt.UTC(), UpdatedAt: o.UpdatedAt.UTC(),
	}
}

type PageResponse struct {
	Items      []OrderResponse `json:"items"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	Total      int64           `json:"total"`
	TotalPages int             `json:"totalPages"`
}

func NewPageResponse(p Page) PageResponse {
	items := make([]OrderResponse, len(p.Items))
	for i, o := range p.Items {
		items[i] = NewOrderResponse(o)
	}
	return PageResponse{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages}
}

// ---- seller stats (6c), mirrored in frontend/src/types/seller.ts -------------

type PeriodMoneyResponse struct {
	Last30Days string `json:"last30Days"`
	AllTime    string `json:"allTime"`
}

type PeriodCountResponse struct {
	Last30Days int64 `json:"last30Days"`
	AllTime    int64 `json:"allTime"`
}

type TopProductResponse struct {
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Units     int64  `json:"units"`
	Revenue   string `json:"revenue"`
}

type StoreRatingResponse struct {
	Average float64 `json:"average"`
	Count   int32   `json:"count"`
}

type SellerStatsResponse struct {
	// Every order status as a key, 0 when the store has none.
	OrdersByStatus map[string]int64     `json:"ordersByStatus"`
	Revenue        PeriodMoneyResponse  `json:"revenue"` // items only, no shipping
	UnitsSold      PeriodCountResponse  `json:"unitsSold"`
	TopProducts    []TopProductResponse `json:"topProducts"` // last 30 days, at most 5
	Rating         StoreRatingResponse  `json:"rating"`
	Since          time.Time            `json:"since"` // start of the 30-day window
}

func NewSellerStatsResponse(s SellerStats) SellerStatsResponse {
	top := make([]TopProductResponse, len(s.TopProducts))
	for i, p := range s.TopProducts {
		top[i] = TopProductResponse{
			ProductID: p.ProductID.String(), Name: p.Name, Status: p.Status,
			Units: p.Units, Revenue: p.Revenue.StringFixed(2),
		}
	}
	return SellerStatsResponse{
		OrdersByStatus: s.OrdersByStatus,
		Revenue:        PeriodMoneyResponse{Last30Days: s.RevenueWindow.StringFixed(2), AllTime: s.RevenueAll.StringFixed(2)},
		UnitsSold:      PeriodCountResponse{Last30Days: s.UnitsWindow, AllTime: s.UnitsAll},
		TopProducts:    top,
		Rating:         StoreRatingResponse{Average: s.RatingAverage, Count: s.RatingCount},
		Since:          s.Since.UTC(),
	}
}
