package reviews

import (
	"time"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/rating"
)

// JSON contracts. Mirrored in frontend/src/types/reviews.ts — keep in sync.

// CreateReviewRequest is POST /api/v1/reviews. Unknown fields are rejected by
// request.BindJSON, so a client cannot send productId, userId or status.
type CreateReviewRequest struct {
	OrderItemID string `json:"orderItemId"`
	Rating      int    `json:"rating"`
	Comment     string `json:"comment"`
}

// UpdateReviewRequest is PATCH /api/v1/reviews/:id. Omitted = unchanged.
type UpdateReviewRequest struct {
	Rating  *int    `json:"rating"`
	Comment *string `json:"comment"`
}

type ReviewResponse struct {
	ID          string    `json:"id"`
	OrderItemID string    `json:"orderItemId"`
	ProductID   string    `json:"productId"`
	Rating      int16     `json:"rating"`
	Comment     string    `json:"comment"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func NewReviewResponse(r Review) ReviewResponse {
	return ReviewResponse{
		ID: r.ID.String(), OrderItemID: r.OrderItemID.String(), ProductID: r.ProductID.String(),
		Rating: r.Rating, Comment: r.Comment, Status: r.Status,
		CreatedAt: r.CreatedAt.UTC(), UpdatedAt: r.UpdatedAt.UTC(),
	}
}

type ImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ItemRefResponse struct {
	OrderItemID string         `json:"orderItemId"`
	OrderID     string         `json:"orderId"`
	OrderNumber string         `json:"orderNumber"`
	ProductID   string         `json:"productId"`
	ProductSlug string         `json:"productSlug"`
	StoreSlug   string         `json:"storeSlug"`
	ProductName string         `json:"productName"`
	VariantName string         `json:"variantName"`
	Image       *ImageResponse `json:"image"`
}

func newItemRef(r ItemRef) ItemRefResponse {
	out := ItemRefResponse{
		OrderItemID: r.OrderItemID.String(), OrderID: r.OrderID.String(), OrderNumber: r.OrderNumber,
		ProductID: r.ProductID.String(), ProductSlug: r.ProductSlug, StoreSlug: r.StoreSlug,
		ProductName: r.ProductName, VariantName: r.VariantName,
	}
	if r.Image != nil {
		out.Image = &ImageResponse{URL: r.Image.URL, Alt: r.Image.Alt}
	}
	return out
}

// ---- public -----------------------------------------------------------------

// PublicReviewResponse never contains the reviewer's id or email.
type PublicReviewResponse struct {
	ID           string    `json:"id"`
	Rating       int16     `json:"rating"`
	Comment      string    `json:"comment"`
	ReviewerName string    `json:"reviewerName"`
	VariantName  string    `json:"variantName"`
	Edited       bool      `json:"edited"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SummaryResponse struct {
	Average float64 `json:"average"`
	Count   int32   `json:"count"`
	// distribution["5"] = number of 5-star reviews; every key 1..5 is present.
	Distribution map[string]int32 `json:"distribution"`
}

type ProductReviewsResponse struct {
	Summary    SummaryResponse        `json:"summary"`
	Items      []PublicReviewResponse `json:"items"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"pageSize"`
	Total      int64                  `json:"total"`
	TotalPages int                    `json:"totalPages"`
}

func NewProductReviewsResponse(s Summary, p Page[PublicReview]) ProductReviewsResponse {
	dist := make(map[string]int32, MaxRating)
	for i, n := range s.Distribution {
		dist[string(rune('1'+i))] = n
	}
	items := make([]PublicReviewResponse, len(p.Items))
	for i, r := range p.Items {
		items[i] = PublicReviewResponse{
			ID: r.ID.String(), Rating: r.Rating, Comment: r.Comment, ReviewerName: r.ReviewerName,
			VariantName: r.VariantName, Edited: r.Edited, CreatedAt: r.CreatedAt.UTC(),
		}
	}
	return ProductReviewsResponse{
		Summary: SummaryResponse{Average: rating.Average(s.Count, s.Total), Count: s.Count, Distribution: dist},
		Items:   items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages,
	}
}

// ---- the caller's own ---------------------------------------------------------

type MyReviewResponse struct {
	ReviewResponse
	Item ItemRefResponse `json:"item"`
}

type PendingItemResponse struct {
	ItemRefResponse
	StoreName   string    `json:"storeName"`
	DeliveredAt time.Time `json:"deliveredAt"`
}

type PageResponse[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

func NewMyReviewsResponse(p Page[MyReview]) PageResponse[MyReviewResponse] {
	items := make([]MyReviewResponse, len(p.Items))
	for i, r := range p.Items {
		items[i] = MyReviewResponse{ReviewResponse: NewReviewResponse(r.Review), Item: newItemRef(r.Item)}
	}
	return PageResponse[MyReviewResponse]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages}
}

func NewPendingResponse(p Page[PendingItem]) PageResponse[PendingItemResponse] {
	items := make([]PendingItemResponse, len(p.Items))
	for i, it := range p.Items {
		items[i] = PendingItemResponse{ItemRefResponse: newItemRef(it.ItemRef), StoreName: it.StoreName, DeliveredAt: it.DeliveredAt.UTC()}
	}
	return PageResponse[PendingItemResponse]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages}
}
