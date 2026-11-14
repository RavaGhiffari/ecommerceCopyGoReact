// Package orders owns the per-store orders created by a checkout: their
// numbers, snapshots, status history, and the buyer's read access.
//
// An order is HISTORY. Product names, SKUs, prices, the store name and the
// delivery address are copied onto the order when it is created, so later
// edits to products, stores or addresses never change what was bought.
//
// Creating orders happens only inside the checkout transaction (Create takes
// the caller's tx). Every status change goes through Repository.Transition
// and the table in transitions.go.
package orders

import (
	"crypto/rand"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// The 7 order statuses (CLAUDE.md §7); allowed changes are in transitions.go.
const (
	StatusPendingPayment = "pending_payment"
	StatusPaid           = "paid"
	StatusProcessing     = "processing"
	StatusShipped        = "shipped"
	StatusDelivered      = "delivered"
	StatusCancelled      = "cancelled"
	StatusRefunded       = "refunded"
)

// Statuses lists every order status (for validating filters and requests).
var Statuses = []string{
	StatusPendingPayment, StatusPaid, StatusProcessing, StatusShipped,
	StatusDelivered, StatusCancelled, StatusRefunded,
}

var (
	ErrNotFound = errors.New("order not found")
	ErrNoStore  = errors.New("seller has no store")
)

// ShippingAddress is the address snapshot stored on the order.
type ShippingAddress struct {
	RecipientName string
	Phone         string
	AddressLine   string
	City          string
	Province      string
	PostalCode    string
}

type Image struct {
	URL string
	Alt string
}

type Item struct {
	ID          uuid.UUID
	VariantID   uuid.UUID
	ProductID   uuid.UUID
	ProductSlug string // current, for links only
	ProductName string // snapshot
	VariantName string // snapshot
	SKU         string // snapshot
	UnitPrice   decimal.Decimal
	Quantity    int32
	LineTotal   decimal.Decimal
	Image       *Image // current first image, for thumbnails only
}

type HistoryEntry struct {
	FromStatus *string // nil for the creation
	ToStatus   string
	Note       string
	CreatedAt  time.Time
}

type Order struct {
	ID             uuid.UUID
	CheckoutID     uuid.UUID
	OrderNumber    string
	Status         string
	StoreID        uuid.UUID
	StoreSlug      string
	StoreName      string // snapshot
	ShippingMethod string
	ShippingFee    decimal.Decimal
	ItemsSubtotal  decimal.Decimal
	Total          decimal.Decimal
	ShipTo         ShippingAddress
	Items          []Item
	History        []HistoryEntry // only filled for the detail view
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Locked is an order row locked FOR UPDATE by the current transaction.
type Locked struct {
	ID         uuid.UUID
	CheckoutID uuid.UUID
	Status     string
}

// StockLine is a quantity of one variant held by an order or a checkout.
type StockLine struct {
	VariantID uuid.UUID
	ProductID uuid.UUID
	Quantity  int32
}

// VariantIDs lists the variants of stock lines (already in variant order).
func VariantIDs(lines []StockLine) []uuid.UUID {
	ids := make([]uuid.UUID, len(lines))
	for i, l := range lines {
		ids[i] = l.VariantID
	}
	return ids
}

// Page is one page of the buyer's order list.
type Page struct {
	Items      []Order
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

// ---- order numbers ------------------------------------------------------------

// crockford is Crockford's base-32 alphabet: digits and capitals without
// I, L, O and U, so a number read aloud or typed by hand is not ambiguous.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// wib is Western Indonesia Time (UTC+7). A fixed zone instead of
// time.LoadLocation("Asia/Jakarta"): no tzdata needed in the container, and
// Indonesia has no daylight saving time.
var wib = time.FixedZone("WIB", 7*60*60)

// NewOrderNumber returns e.g. "PK-20261114-7KQ2M9": the date in WIB and 6
// random characters (32^6 ≈ 1.07 billion per day). It is not guaranteed to be
// unique; the UNIQUE constraint decides, and Create retries on a collision.
func NewOrderNumber(now time.Time) string {
	var b [6]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error (Go 1.24+)
	suffix := make([]byte, len(b))
	for i, v := range b {
		// 256 is a multiple of 32, so v%32 is uniform (no modulo bias).
		suffix[i] = crockford[v%32]
	}
	return "PK-" + now.In(wib).Format("20060102") + "-" + string(suffix)
}
