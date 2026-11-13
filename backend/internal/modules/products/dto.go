package products

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/categories"
)

// JSON contracts. Mirrored in frontend/src/types/marketplace.ts and
// frontend/src/types/seller.ts — keep them in sync.
//
// Money is sent as a decimal STRING with two decimals ("349000.00"), never a
// JSON number: JSON numbers become floating point in JavaScript, and money
// must not silently lose precision on the way through.
func money(d decimal.Decimal) string { return d.StringFixed(2) }

type ImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type StoreRefResponse struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type ratingResponse struct {
	Average float64 `json:"average"` // reviews arrive in Phase 6; 0 until then
	Count   int     `json:"count"`
}

type ProductSummaryResponse struct {
	ID           string           `json:"id"`
	Slug         string           `json:"slug"`
	Name         string           `json:"name"`
	CategoryID   string           `json:"categoryId"`
	Images       []ImageResponse  `json:"images"`
	LowestPrice  string           `json:"lowestPrice"`
	HighestPrice string           `json:"highestPrice"`
	Rating       ratingResponse   `json:"rating"`
	SoldCount    int32            `json:"soldCount"`
	InStock      bool             `json:"inStock"`
	CreatedAt    time.Time        `json:"createdAt"`
	Store        StoreRefResponse `json:"store"`
}

func NewSummaryResponse(s Summary) ProductSummaryResponse {
	images := []ImageResponse{}
	if s.Image != nil {
		images = append(images, ImageResponse(*s.Image))
	}
	return ProductSummaryResponse{
		ID: s.ID.String(), Slug: s.Slug, Name: s.Name, CategoryID: s.CategoryID.String(),
		Images: images, LowestPrice: money(s.LowestPrice), HighestPrice: money(s.HighestPrice),
		SoldCount: s.SoldCount, InStock: s.InStock, CreatedAt: s.CreatedAt,
		Store: StoreRefResponse{ID: s.Store.ID.String(), Slug: s.Store.Slug, Name: s.Store.Name, Location: s.Store.Location},
	}
}

type VariantResponse struct {
	ID    string `json:"id"`
	SKU   string `json:"sku"`
	Name  string `json:"name"`
	Price string `json:"price"`
	Stock int32  `json:"stock"`
}

func newVariants(vs []Variant) []VariantResponse {
	out := make([]VariantResponse, len(vs))
	for i, v := range vs {
		out[i] = VariantResponse{ID: v.ID.String(), SKU: v.SKU, Name: v.Name, Price: money(v.Price), Stock: v.Stock}
	}
	return out
}

func newImages(imgs []Image) []ImageResponse {
	out := make([]ImageResponse, len(imgs))
	for i, img := range imgs {
		out[i] = ImageResponse(img)
	}
	return out
}

type ProductDetailResponse struct {
	ProductSummaryResponse
	Description string                      `json:"description"`
	Category    categories.CategoryResponse `json:"category"`
	Variants    []VariantResponse           `json:"variants"`
}

func NewDetailResponse(d Detail) ProductDetailResponse {
	r := ProductDetailResponse{
		ProductSummaryResponse: NewSummaryResponse(d.Summary),
		Description:            d.Description,
		Category:               categories.NewCategoryResponse(d.Category),
		Variants:               newVariants(d.Variants),
	}
	r.Images = newImages(d.Images) // all images, not only the first
	return r
}

type PageResponse struct {
	Items      []ProductSummaryResponse `json:"items"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"pageSize"`
	Total      int64                    `json:"total"`
	TotalPages int                      `json:"totalPages"`
}

func NewPageResponse(p Page) PageResponse {
	items := make([]ProductSummaryResponse, len(p.Items))
	for i, s := range p.Items {
		items[i] = NewSummaryResponse(s)
	}
	return PageResponse{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages}
}

// ---- seller -----------------------------------------------------------------

type SellerProductResponse struct {
	ID          string            `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	CategoryID  string            `json:"categoryId"`
	Images      []ImageResponse   `json:"images"`
	Variants    []VariantResponse `json:"variants"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func NewSellerProductResponse(p SellerProduct) SellerProductResponse {
	return SellerProductResponse{
		ID: p.ID.String(), Slug: p.Slug, Name: p.Name, Description: p.Description, Status: p.Status,
		CategoryID: p.CategoryID.String(), Images: newImages(p.Images), Variants: newVariants(p.Variants),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

type SellerProductSummaryResponse struct {
	ID           string    `json:"id"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	CategoryID   string    `json:"categoryId"`
	VariantCount int32     `json:"variantCount"`
	TotalStock   int32     `json:"totalStock"`
	LowestPrice  string    `json:"lowestPrice"`
	HighestPrice string    `json:"highestPrice"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewSellerSummaryResponse(s SellerSummary) SellerProductSummaryResponse {
	return SellerProductSummaryResponse{
		ID: s.ID.String(), Slug: s.Slug, Name: s.Name, Status: s.Status, CategoryID: s.CategoryID.String(),
		VariantCount: s.VariantCount, TotalStock: s.TotalStock,
		LowestPrice: money(s.LowestPrice), HighestPrice: money(s.HighestPrice), UpdatedAt: s.UpdatedAt,
	}
}

type ImageRequest struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

// VariantRequest: price accepts "349000", "349000.50" or a JSON number; it is
// parsed straight into an exact decimal, never through float64.
type VariantRequest struct {
	Name  string           `json:"name"`
	SKU   string           `json:"sku"`
	Price *decimal.Decimal `json:"price"`
	Stock *int32           `json:"stock"`
}

type CreateProductRequest struct {
	Name        string           `json:"name"`
	Slug        string           `json:"slug"`
	Description string           `json:"description"`
	CategoryID  *uuid.UUID       `json:"categoryId"`
	Images      []ImageRequest   `json:"images"`
	Variants    []VariantRequest `json:"variants"`
}

type UpdateProductRequest struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	CategoryID  *uuid.UUID      `json:"categoryId"`
	Status      *string         `json:"status"`
	Images      *[]ImageRequest `json:"images"`
}

type UpdateVariantRequest struct {
	Name  *string          `json:"name"`
	Price *decimal.Decimal `json:"price"`
}

type SetStockRequest struct {
	Quantity *int32 `json:"quantity"`
	Reason   string `json:"reason"`
}

type StockResponse struct {
	VariantID         string `json:"variantId"`
	AvailableQuantity int32  `json:"availableQuantity"`
}

func toImageInputs(in []ImageRequest) []ImageInput {
	out := make([]ImageInput, len(in))
	for i, img := range in {
		out[i] = ImageInput(img)
	}
	return out
}

func toVariantInput(v VariantRequest) VariantInput {
	return VariantInput(v)
}
