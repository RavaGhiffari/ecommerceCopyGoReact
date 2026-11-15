package stores

import (
	"time"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/rating"
)

type ratingResponse struct {
	Average float64 `json:"average"` // display value, not money
	Count   int     `json:"count"`
}

// PublicStoreResponse is GET /api/v1/stores/:slug. Mirrors StoreDetail in
// frontend/src/types/marketplace.ts. The rating covers every review of every
// product of the store (Phase 6a).
type PublicStoreResponse struct {
	ID                 string         `json:"id"`
	Slug               string         `json:"slug"`
	Name               string         `json:"name"`
	Location           string         `json:"location"`
	Description        string         `json:"description"`
	Rating             ratingResponse `json:"rating"`
	JoinedAt           time.Time      `json:"joinedAt"`
	ActiveProductCount int32          `json:"activeProductCount"`
}

func NewPublicStoreResponse(s Store) PublicStoreResponse {
	return PublicStoreResponse{
		ID: s.ID.String(), Slug: s.Slug, Name: s.Name, Location: s.Location, Description: s.Description,
		JoinedAt: s.CreatedAt, ActiveProductCount: s.ActiveProductCount,
		Rating: ratingResponse{Average: rating.Average(s.RatingCount, s.RatingTotal), Count: int(s.RatingCount)},
	}
}

// SellerStoreResponse is what the owner sees about their own store.
type SellerStoreResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

func NewSellerStoreResponse(s Store) SellerStoreResponse {
	return SellerStoreResponse{
		ID: s.ID.String(), Slug: s.Slug, Name: s.Name, Description: s.Description,
		Location: s.Location, Status: s.Status, CreatedAt: s.CreatedAt,
	}
}

// OpenStoreRequest is POST /api/v1/seller/store.
type OpenStoreRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Location    string `json:"location"`
}

// UpdateStoreRequest is PATCH /api/v1/seller/store. Omitted = unchanged.
type UpdateStoreRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Location    *string `json:"location"`
}
