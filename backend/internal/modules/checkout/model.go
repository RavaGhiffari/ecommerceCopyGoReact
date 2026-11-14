// Package checkout turns the buyer's whole cart into one checkout with one
// order per store, in a single PostgreSQL transaction (PLAN.md §5.4).
//
// Rules (approved Phase 5 decisions):
//   - Q3: the whole cart is checked out; any line with an issue blocks it.
//   - Q2/Q11: a shipping method per store order, flat fees from config;
//     Same Day only when the store city equals the delivery city.
//   - Q4: one simulated payment per checkout, covering all its orders.
//   - Q5: the checkout holds its stock (reservation) until it expires.
//   - Q6/Q12: the Idempotency-Key header makes a repeated submit return the
//     first checkout; the same key with a different body is a 409.
//   - Prices and fees always come from the database and config, never from
//     the request.
package checkout

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/shipping"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// Why a line cannot be bought as it is. Same values as the cart, plus
// own_product (the buyer opened a store after adding the item).
const (
	IssueUnavailable       = "unavailable"
	IssueOutOfStock        = "out_of_stock"
	IssueInsufficientStock = "insufficient_stock"
	IssueOwnProduct        = "own_product"
)

const (
	StatusPendingPayment = "pending_payment"
	ReferenceCheckout    = "checkout" // inventory_movements.reference_type
)

var (
	ErrCartEmpty       = errors.New("cart is empty")
	ErrAddressNotFound = errors.New("address not found")
	ErrNotFound        = errors.New("checkout not found")
	ErrKeyReused       = errors.New("idempotency key was used for a different request")
)

// IssuesError: some lines cannot be bought. Nothing was written.
type IssuesError struct {
	Lines []Line // only the lines with an issue
}

func (e *IssuesError) Error() string {
	return fmt.Sprintf("%d cart line(s) cannot be bought", len(e.Lines))
}

// MethodNotAllowedError: a chosen shipping method is not offered for that
// store and destination (e.g. Same Day to another city). Keys are
// "shipping.<storeId>", like validation errors.
type MethodNotAllowedError struct {
	Fields map[string]string
}

func (e *MethodNotAllowedError) Error() string { return "shipping method not allowed" }

// Input is the body of POST /checkout/quote and POST /checkout.
type Input struct {
	AddressID uuid.UUID
	// Chosen method per store id. For a quote a missing store means
	// "regular"; for placing an order every store needs a choice.
	Shipping map[uuid.UUID]shipping.Method
}

// ParseInput validates the raw request fields. Store ids that are not in the
// cart are reported later, when the cart is known.
func ParseInput(addressID *uuid.UUID, raw map[string]string) (Input, error) {
	errs := validation.Errors{}
	in := Input{Shipping: map[uuid.UUID]shipping.Method{}}
	if addressID == nil {
		errs.Add("addressId", "is required")
	} else {
		in.AddressID = *addressID
	}
	for key, method := range raw {
		id, err := uuid.Parse(key)
		if err != nil {
			errs.Add("shipping."+key, "is not a store in your cart")
			continue
		}
		if !shipping.Valid(method) {
			errs.Add("shipping."+key, "must be regular, express or same_day")
			continue
		}
		in.Shipping[id] = shipping.Method(method)
	}
	return in, errs.Err()
}

// Hash is the request fingerprint stored with the checkout (decision Q12).
// It is computed from the PARSED input in a fixed order, so the same request
// sent with its JSON keys in another order still matches.
func (in Input) Hash() []byte {
	var b strings.Builder
	b.WriteString("address=" + in.AddressID.String() + "\n")
	stores := make([]string, 0, len(in.Shipping))
	for id, m := range in.Shipping {
		stores = append(stores, id.String()+"="+string(m))
	}
	sort.Strings(stores)
	for _, s := range stores {
		b.WriteString("shipping:" + s + "\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return sum[:]
}

// ---- quote --------------------------------------------------------------------

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
	ItemID            uuid.UUID // cart item id
	VariantID         uuid.UUID
	VariantName       string
	SKU               string
	ProductID         uuid.UUID
	ProductSlug       string
	ProductName       string
	Image             *Image
	UnitPrice         decimal.Decimal // current price from the database
	Quantity          int32
	LineTotal         decimal.Decimal
	AvailableQuantity int32
	Issue             string
}

type Group struct {
	Store       StoreRef
	Lines       []Line
	Options     []shipping.Option // methods offered for this destination
	Method      shipping.Method   // chosen (or "regular" in a quote)
	ShippingFee decimal.Decimal
	Subtotal    decimal.Decimal // lines without an issue
	Total       decimal.Decimal // Subtotal + ShippingFee
}

type Quote struct {
	Address       addresses.Address
	Groups        []Group
	ItemsTotal    decimal.Decimal
	ShippingTotal decimal.Decimal
	GrandTotal    decimal.Decimal
	HasIssues     bool
}

// IssueLines returns every line that cannot be bought.
func (q Quote) IssueLines() []Line {
	var out []Line
	for _, g := range q.Groups {
		for _, l := range g.Lines {
			if l.Issue != "" {
				out = append(out, l)
			}
		}
	}
	return out
}

// lineRow is what the repository reads (kept free of database types).
type lineRow struct {
	ItemID            uuid.UUID
	Quantity          int32
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
	StoreOwnerID      uuid.UUID
	AvailableQuantity int32
	Image             *Image
}

func classify(r lineRow, buyerID uuid.UUID) string {
	switch {
	case r.StoreOwnerID == buyerID:
		return IssueOwnProduct
	case r.ProductStatus != "active" || r.StoreStatus != "active":
		return IssueUnavailable
	case r.AvailableQuantity <= 0:
		return IssueOutOfStock
	case r.Quantity > r.AvailableQuantity:
		return IssueInsufficientStock
	default:
		return ""
	}
}

// buildQuote groups the cart rows by store (in query order), prices every
// line from the database price, applies the shipping choice and adds it all
// up with exact decimals. Pure: no database, no clock.
//
// strict = placing an order: every store in the cart needs a method.
// Otherwise (a quote) a store without a choice gets "regular".
func buildQuote(rows []lineRow, buyerID uuid.UUID, addr addresses.Address, choice map[uuid.UUID]shipping.Method,
	rates shipping.Rates, strict bool) (Quote, error) {

	q := Quote{Address: addr, Groups: []Group{}, ItemsTotal: decimal.Zero, ShippingTotal: decimal.Zero}
	index := map[uuid.UUID]int{}

	for _, r := range rows {
		line := Line{
			ItemID: r.ItemID, VariantID: r.VariantID, VariantName: r.VariantName, SKU: r.SKU,
			ProductID: r.ProductID, ProductSlug: r.ProductSlug, ProductName: r.ProductName, Image: r.Image,
			UnitPrice: r.Price, Quantity: r.Quantity,
			LineTotal:         r.Price.Mul(decimal.NewFromInt32(r.Quantity)),
			AvailableQuantity: r.AvailableQuantity,
			Issue:             classify(r, buyerID),
		}
		i, ok := index[r.Store.ID]
		if !ok {
			i = len(q.Groups)
			index[r.Store.ID] = i
			q.Groups = append(q.Groups, Group{Store: r.Store, Subtotal: decimal.Zero})
		}
		g := &q.Groups[i]
		g.Lines = append(g.Lines, line)
		if line.Issue != "" {
			q.HasIssues = true
			continue
		}
		g.Subtotal = g.Subtotal.Add(line.LineTotal)
	}

	// The shipping choice may only name stores that are in the cart.
	errs := validation.Errors{}
	for storeID := range choice {
		if _, ok := index[storeID]; !ok {
			errs.Add("shipping."+storeID.String(), "is not a store in your cart")
		}
	}
	notAllowed := map[string]string{}

	for i := range q.Groups {
		g := &q.Groups[i]
		g.Options = rates.Options(g.Store.Location, addr.City)
		key := "shipping." + g.Store.ID.String()

		m, chosen := choice[g.Store.ID]
		if !chosen {
			if strict {
				errs.Add(key, "is required")
				continue
			}
			m = shipping.Regular
		}
		if !rates.Allowed(m, g.Store.Location, addr.City) {
			notAllowed[key] = fmt.Sprintf("%s is not available from %s to %s", m, g.Store.Location, addr.City)
			continue
		}
		fee, _ := rates.Fee(m)
		g.Method, g.ShippingFee = m, fee
		g.Total = g.Subtotal.Add(fee)
		q.ItemsTotal = q.ItemsTotal.Add(g.Subtotal)
		q.ShippingTotal = q.ShippingTotal.Add(fee)
	}

	if err := errs.Err(); err != nil {
		return Quote{}, err
	}
	if len(notAllowed) > 0 {
		return Quote{}, &MethodNotAllowedError{Fields: notAllowed}
	}
	q.GrandTotal = q.ItemsTotal.Add(q.ShippingTotal)
	return q, nil
}

// ---- placed checkout -------------------------------------------------------------

type Payment struct {
	ID        uuid.UUID
	Reference string
	Provider  string
	Amount    decimal.Decimal
	Status    string
	ExpiresAt time.Time
	PaidAt    *time.Time
}

// Checkout is a placed checkout as the buyer sees it (orders are loaded
// through the orders package).
type Checkout struct {
	ID            uuid.UUID
	Status        string
	ItemsTotal    decimal.Decimal
	ShippingTotal decimal.Decimal
	GrandTotal    decimal.Decimal
	ExpiresAt     time.Time
	PaidAt        *time.Time
	CancelledAt   *time.Time
	CreatedAt     time.Time
	Payment       Payment
}
