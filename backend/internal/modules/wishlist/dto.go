package wishlist

import "time"

// JSON contracts. Mirrored in frontend/src/types/wishlist.ts — keep in sync.
// Money is a decimal string with two decimals ("349000.00").

type ImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type StoreRefResponse struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type ItemResponse struct {
	ProductID   string           `json:"productId"`
	Slug        string           `json:"slug"`
	Name        string           `json:"name"`
	Image       *ImageResponse   `json:"image"`
	LowestPrice string           `json:"lowestPrice"`
	Store       StoreRefResponse `json:"store"`
	AddedAt     time.Time        `json:"addedAt"`
}

type PageResponse struct {
	Items      []ItemResponse `json:"items"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	Total      int64          `json:"total"`
	TotalPages int            `json:"totalPages"`
}

func NewPageResponse(p Page) PageResponse {
	items := make([]ItemResponse, len(p.Items))
	for i, it := range p.Items {
		items[i] = ItemResponse{
			ProductID: it.ProductID.String(), Slug: it.Slug, Name: it.Name,
			LowestPrice: it.LowestPrice.StringFixed(2),
			Store:       StoreRefResponse{Slug: it.StoreSlug, Name: it.StoreName},
			AddedAt:     it.AddedAt.UTC(),
		}
		if it.Image != nil {
			items[i].Image = &ImageResponse{URL: it.Image.URL, Alt: it.Image.Alt}
		}
	}
	return PageResponse{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages}
}

// IDsResponse is GET /api/v1/me/wishlist/ids.
type IDsResponse struct {
	ProductIDs []string `json:"productIds"`
}
