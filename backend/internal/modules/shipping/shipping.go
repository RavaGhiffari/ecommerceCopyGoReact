// Package shipping is the simulated courier: which methods a store order may
// use and what each costs (approved decision Q2).
//
// Pure functions only, no database: the rules are easy to read and to unit
// test, and the checkout service calls them while computing a quote.
//
// Rules:
//   - Every store order picks one method; the fee is a flat amount per
//     store order (not per item, not by weight).
//   - Same Day is only offered when the store's city and the delivery city
//     are the same (decision Q11: normalised text compare, because both are
//     free text; "Jakarta Selatan" is not "Jakarta").
package shipping

import (
	"strings"

	"github.com/shopspring/decimal"
)

type Method string

const (
	Regular Method = "regular"
	Express Method = "express"
	SameDay Method = "same_day"
)

// All lists the methods in display order (cheapest first).
var All = []Method{Regular, Express, SameDay}

// Valid reports whether s names a known method (used to validate requests).
func Valid(s string) bool {
	for _, m := range All {
		if string(m) == s {
			return true
		}
	}
	return false
}

// Rates are the flat fees per store order. They come from config
// (SHIPPING_FEE_*), so they are never taken from the client.
type Rates struct {
	Regular decimal.Decimal
	Express decimal.Decimal
	SameDay decimal.Decimal
}

// DefaultRates are the approved amounts in rupiah.
func DefaultRates() Rates {
	return Rates{
		Regular: decimal.NewFromInt(15_000),
		Express: decimal.NewFromInt(30_000),
		SameDay: decimal.NewFromInt(45_000),
	}
}

// Fee returns the fee of a method. Unknown methods return false.
func (r Rates) Fee(m Method) (decimal.Decimal, bool) {
	switch m {
	case Regular:
		return r.Regular, true
	case Express:
		return r.Express, true
	case SameDay:
		return r.SameDay, true
	}
	return decimal.Zero, false
}

// Option is one method the buyer may pick for a store order, with its fee.
type Option struct {
	Method Method
	Fee    decimal.Decimal
}

// Options returns the methods allowed for a store in storeCity delivering to
// destinationCity, in display order.
func (r Rates) Options(storeCity, destinationCity string) []Option {
	out := make([]Option, 0, len(All))
	for _, m := range All {
		if m == SameDay && !SameCity(storeCity, destinationCity) {
			continue
		}
		fee, _ := r.Fee(m)
		out = append(out, Option{Method: m, Fee: fee})
	}
	return out
}

// Allowed reports whether m may be used for that store and destination.
func (r Rates) Allowed(m Method, storeCity, destinationCity string) bool {
	for _, o := range r.Options(storeCity, destinationCity) {
		if o.Method == m {
			return true
		}
	}
	return false
}

// SameCity compares two free-text city names ignoring case and extra spaces.
func SameCity(a, b string) bool {
	na, nb := normalizeCity(a), normalizeCity(b)
	return na != "" && na == nb
}

func normalizeCity(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
