// Package stores manages seller stores: one store per seller.
package stores

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound        = errors.New("store not found")
	ErrAlreadyHasStore = errors.New("user already owns a store")
	ErrSlugTaken       = errors.New("store slug already used")
)

type Store struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	Slug        string
	Description string
	Location    string
	Status      string // active | suspended
	CreatedAt   time.Time
	// Only filled by the public lookup.
	ActiveProductCount int32
}
