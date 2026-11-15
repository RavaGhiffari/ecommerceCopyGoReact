// Package reviews lets buyers rate what they bought (Phase 6a).
//
// Rules (PLAN.md §6.0 Q1–Q8):
//   - Only an order item of the caller's own DELIVERED order can be reviewed,
//     once (reviews.order_item_id is UNIQUE).
//   - The author may edit rating and comment; nobody deletes a review.
//   - products.rating_count / rating_total are updated in the same
//     transaction as every review write, with an atomic UPDATE.
//   - The public list shows a masked reviewer name, never an email or id.
package reviews

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

var (
	ErrItemNotFound    = errors.New("order item not found")
	ErrNotAllowed      = errors.New("order item is not reviewable")
	ErrExists          = errors.New("order item already reviewed")
	ErrNotFound        = errors.New("review not found")
	ErrProductNotFound = errors.New("product not found")
)

const (
	StatusVisible = "visible"
	StatusHidden  = "hidden"

	MinRating        = 1
	MaxRating        = 5
	MaxCommentLength = 1000 // characters (runes), matches the DB CHECK

	DefaultPageSize = 10
	MaxPageSize     = 50
)

type Review struct {
	ID          uuid.UUID
	OrderItemID uuid.UUID
	ProductID   uuid.UUID
	Rating      int16
	Comment     string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PublicReview is one row of a product's review list.
type PublicReview struct {
	ID           uuid.UUID
	Rating       int16
	Comment      string
	ReviewerName string // already masked
	VariantName  string
	CreatedAt    time.Time
	Edited       bool
}

// Summary is the rating block of a product page.
type Summary struct {
	Count        int32
	Total        int32
	Distribution [MaxRating]int32 // index 0 = 1 star … 4 = 5 stars
}

type Image struct {
	URL string
	Alt string
}

// ItemRef identifies what a review is about, for links in "my reviews" and
// "waiting for review".
type ItemRef struct {
	OrderItemID uuid.UUID
	OrderID     uuid.UUID
	OrderNumber string
	ProductID   uuid.UUID
	ProductSlug string // current, for links
	StoreSlug   string
	ProductName string // snapshot
	VariantName string // snapshot
	Image       *Image
}

type MyReview struct {
	Review
	Item ItemRef
}

type PendingItem struct {
	ItemRef
	StoreName   string
	DeliveredAt time.Time
}

type Page[T any] struct {
	Items      []T
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

// Input is a review's editable content, before validation.
type Input struct {
	Rating  int
	Comment string
}

// Clean validates and normalises the input (comment trimmed).
func (in Input) Clean() (Input, error) {
	errs := validation.Errors{}
	if in.Rating < MinRating || in.Rating > MaxRating {
		errs.Add("rating", "must be between 1 and 5")
	}
	in.Comment = strings.TrimSpace(in.Comment)
	if utf8.RuneCountInString(in.Comment) > MaxCommentLength {
		errs.Add("comment", "must be at most 1000 characters")
	}
	return in, errs.Err()
}

// MaskName shows the first name and hides the rest: "Rava Ghiffari" ->
// "Rava G***", "Rava" -> "R***". Reviews are public; the full name is not.
func MaskName(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return "Pembeli"
	}
	if len(words) == 1 {
		r, _ := utf8.DecodeRuneInString(words[0])
		return string(r) + "***"
	}
	r, _ := utf8.DecodeRuneInString(words[1])
	return words[0] + " " + string(r) + "***"
}

func pageCount(total int64, size int) int {
	n := int((total + int64(size) - 1) / int64(size))
	if n == 0 {
		return 1
	}
	return n
}
