// Package categories manages the two-level category tree.
// Anyone can read it; only admins change it.
package categories

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("category not found")
	ErrSlugTaken = errors.New("category slug already used")
	// ErrTooDeep is the database trigger's verdict (see migration 000005);
	// the service normally catches the same mistake first as a validation error.
	ErrTooDeep = errors.New("categories can only be two levels deep")
)

type Category struct {
	ID        uuid.UUID
	ParentID  *uuid.UUID // nil for a top-level category
	Name      string
	Slug      string
	SortOrder int32
}
