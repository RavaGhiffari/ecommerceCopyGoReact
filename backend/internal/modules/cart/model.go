// Package cart manages each customer's shopping cart.
//
// Rules (CLAUDE.md §7 and the approved Phase 4 decisions):
//   - Login required; one cart per user, created on the first add.
//   - A cart is NOT a reservation: nothing here touches `inventory`. Stock is
//     checked on add/update for early feedback and re-checked at checkout.
//   - Quantity per line is 1..min(stock, 99).
//   - Sellers cannot put their own store's products in their cart.
//   - Prices are always the CURRENT variant price; history is snapshotted only
//     when an order is created (Phase 5).
//   - Lines that became unbuyable (archived, suspended store, stock dropped)
//     stay in the cart with an `issue` instead of disappearing silently, and
//     are left out of the subtotal.
package cart

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const MaxQuantityPerLine = 99

// Why a line cannot be bought as it is. Empty = fine.
const (
	IssueUnavailable       = "unavailable"        // product archived or store suspended
	IssueOutOfStock        = "out_of_stock"       // no stock left at all
	IssueInsufficientStock = "insufficient_stock" // less stock than the quantity in the cart
)

var (
	ErrItemNotFound       = errors.New("cart item not found")
	ErrVariantNotFound    = errors.New("variant not found")
	ErrVariantUnavailable = errors.New("variant is not for sale")
	ErrOutOfStock         = errors.New("variant is out of stock")
	ErrOwnProduct         = errors.New("cannot buy from your own store")
)

// ErrExceedsStock: the requested total is more than the current stock.
type ErrExceedsStock struct {
	Available int32 // current stock
	InCart    int32 // already in the cart before this request (0 for updates)
}

func (e *ErrExceedsStock) Error() string {
	return fmt.Sprintf("only %d available (%d already in cart)", e.Available, e.InCart)
}

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

type Line struct {
	ID                uuid.UUID
	Quantity          int32
	VariantID         uuid.UUID
	VariantName       string
	SKU               string
	ProductID         uuid.UUID
	ProductSlug       string
	ProductName       string
	Image             *Image
	UnitPrice         decimal.Decimal // current price, not a snapshot
	LineTotal         decimal.Decimal // UnitPrice × Quantity
	AvailableQuantity int32
	Issue             string
}

type StoreGroup struct {
	Store    StoreRef
	Lines    []Line
	Subtotal decimal.Decimal // only lines without an issue
}

type Cart struct {
	Groups    []StoreGroup
	ItemCount int32           // sum of quantities over all lines (header badge)
	Subtotal  decimal.Decimal // only lines without an issue
	HasIssues bool
}

// lineRow is what the repository reads; buildCart turns rows into a Cart.
// Kept free of database types so buildCart is a pure, easily tested function.
type lineRow struct {
	ID                uuid.UUID
	Quantity          int32
	CreatedAt         time.Time
	VariantID         uuid.UUID
	VariantName       string
	SKU               string
	Price             decimal.Decimal
	ProductID         uuid.UUID
	ProductSlug       string
	ProductName       string
	ProductStatus     string
	Store             StoreRef
	StoreStatus       string
	AvailableQuantity int32
	Image             *Image
}

// classify decides whether a line can be bought as it is.
func classify(productStatus, storeStatus string, quantity, available int32) string {
	switch {
	case productStatus != "active" || storeStatus != "active":
		return IssueUnavailable
	case available <= 0:
		return IssueOutOfStock
	case quantity > available:
		return IssueInsufficientStock
	default:
		return ""
	}
}

// buildCart groups rows by store, preserving their order (the query sorts by
// store, then by when each line was added), and computes the money totals
// with exact decimals.
func buildCart(rows []lineRow) Cart {
	c := Cart{Groups: []StoreGroup{}, Subtotal: decimal.Zero}
	index := map[uuid.UUID]int{}

	for _, r := range rows {
		line := Line{
			ID: r.ID, Quantity: r.Quantity, VariantID: r.VariantID, VariantName: r.VariantName, SKU: r.SKU,
			ProductID: r.ProductID, ProductSlug: r.ProductSlug, ProductName: r.ProductName, Image: r.Image,
			UnitPrice:         r.Price,
			LineTotal:         r.Price.Mul(decimal.NewFromInt32(r.Quantity)),
			AvailableQuantity: r.AvailableQuantity,
			Issue:             classify(r.ProductStatus, r.StoreStatus, r.Quantity, r.AvailableQuantity),
		}

		i, ok := index[r.Store.ID]
		if !ok {
			i = len(c.Groups)
			index[r.Store.ID] = i
			c.Groups = append(c.Groups, StoreGroup{Store: r.Store, Subtotal: decimal.Zero})
		}
		g := &c.Groups[i]
		g.Lines = append(g.Lines, line)

		c.ItemCount += line.Quantity
		if line.Issue != "" {
			c.HasIssues = true
			continue
		}
		g.Subtotal = g.Subtotal.Add(line.LineTotal)
		c.Subtotal = c.Subtotal.Add(line.LineTotal)
	}
	return c
}
