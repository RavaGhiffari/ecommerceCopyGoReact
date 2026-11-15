// Package wishlist lets a signed-in user save products for later (practice
// feature, not part of the phase plan).
//
// Rules:
//   - Only public products can be added (active product, active store, at
//     least one variant); anything else is "product not found".
//   - Adding and removing are idempotent: a product is in a wishlist at most
//     once (UNIQUE (user_id, product_id)), and removing a missing one is fine.
//   - Every query filters by the session user, so another user's wishlist
//     cannot be read or changed.
//   - The list only shows products that are still public.
package wishlist

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrProductNotFound = errors.New("product not found")

const (
	DefaultPageSize = 12
	MaxPageSize     = 48
)

type Image struct {
	URL string
	Alt string
}

// Item is one saved product, with what a list row needs.
type Item struct {
	ProductID   uuid.UUID
	Slug        string
	Name        string
	Image       *Image // nil when the product has no image
	LowestPrice decimal.Decimal
	StoreSlug   string
	StoreName   string
	AddedAt     time.Time
}

type Page struct {
	Items      []Item
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}
