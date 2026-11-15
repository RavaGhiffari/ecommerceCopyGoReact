// Package products owns the catalog: the public listing and detail pages, and
// a seller's management of their own products, variants and stock.
//
// Ownership rule: every seller operation first resolves the caller's store
// from the session, and every query on an existing product filters by that
// store_id. A product of another store is indistinguishable from one that does
// not exist (404), so IDs cannot be probed across stores.
package products

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/categories"
)

var (
	ErrNotFound          = errors.New("product not found")
	ErrVariantNotFound   = errors.New("variant not found")
	ErrNoStore           = errors.New("seller has no store")
	ErrSlugTaken         = errors.New("product slug already used in this store")
	ErrSKUTaken          = errors.New("SKU already used")
	ErrVariantNameTaken  = errors.New("variant name already used for this product")
	ErrCategoryNotFound  = errors.New("category not found")
	ErrStoreNotFound     = errors.New("store not found")
	maxPrice             = decimal.RequireFromString("9999999999.99") // NUMERIC(12,2)
	defaultAdjustmentWhy = "manual adjustment"
)

const (
	StatusActive   = "active"
	StatusArchived = "archived"

	SortPopular   = "popular"
	SortNewest    = "newest"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortRating    = "rating"

	DefaultPageSize = 12
	MaxPageSize     = 48
	MaxVariants     = 20
	MaxImages       = 8
)

type StoreRef struct {
	ID       uuid.UUID
	Slug     string
	Name     string
	Location string
}

type Image struct {
	URL string
	Alt string
}

type Variant struct {
	ID    uuid.UUID
	SKU   string
	Name  string
	Price decimal.Decimal
	Stock int32 // available quantity
}

// Summary is one card in a public listing.
type Summary struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	CategoryID   uuid.UUID
	Image        *Image // first image, nil when the product has none
	LowestPrice  decimal.Decimal
	HighestPrice decimal.Decimal
	SoldCount    int32
	RatingCount  int32 // visible reviews (denormalised, Phase 6a)
	RatingTotal  int32 // sum of their stars
	InStock      bool
	CreatedAt    time.Time
	Store        StoreRef
}

// Detail is the public product page.
type Detail struct {
	Summary
	Description string
	Category    categories.Category
	Images      []Image
	Variants    []Variant
}

// Page is one page of a listing plus what the UI needs to paginate.
type Page struct {
	Items      []Summary
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

// SellerProduct is the owner's full view, archived products included.
type SellerProduct struct {
	ID          uuid.UUID
	Slug        string
	Name        string
	Description string
	Status      string
	CategoryID  uuid.UUID
	Images      []Image
	Variants    []Variant
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SellerSummary is one row of the owner's product table.
type SellerSummary struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	Status       string
	CategoryID   uuid.UUID
	VariantCount int32
	TotalStock   int32
	LowestPrice  decimal.Decimal
	HighestPrice decimal.Decimal
	UpdatedAt    time.Time
}
