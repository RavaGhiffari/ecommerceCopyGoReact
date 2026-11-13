package cart

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		product, store string
		qty, available int32
		want           string
	}{
		{"active", "active", 2, 5, ""},
		{"active", "active", 5, 5, ""},
		{"active", "active", 6, 5, IssueInsufficientStock},
		{"active", "active", 1, 0, IssueOutOfStock},
		{"archived", "active", 1, 5, IssueUnavailable},
		{"active", "suspended", 1, 5, IssueUnavailable},
		{"archived", "active", 1, 0, IssueUnavailable}, // unavailable wins over stock
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, classify(tt.product, tt.store, tt.qty, tt.available), "%+v", tt)
	}
}

func row(store StoreRef, price string, qty, available int32, productStatus string) lineRow {
	return lineRow{
		ID: uuid.New(), Quantity: qty, VariantID: uuid.New(), Price: decimal.RequireFromString(price),
		ProductStatus: productStatus, Store: store, StoreStatus: "active", AvailableQuantity: available,
	}
}

func TestBuildCart_GroupsByStoreAndTotalsExactly(t *testing.T) {
	a := StoreRef{ID: uuid.New(), Name: "A"}
	b := StoreRef{ID: uuid.New(), Name: "B"}

	c := buildCart([]lineRow{
		row(a, "0.10", 3, 10, "active"),      // 0.30 — exact with decimals, 0.30000000000000004 with floats
		row(a, "349000.00", 1, 10, "active"), // 349000.00
		row(b, "59000.50", 2, 1, "active"),   // insufficient stock: excluded from totals
		row(b, "1000", 1, 5, "archived"),     // unavailable: excluded from totals
	})

	require.Len(t, c.Groups, 2, "lines are grouped per store, in order")
	assert.Equal(t, a.ID, c.Groups[0].Store.ID)
	assert.Len(t, c.Groups[0].Lines, 2)
	assert.Equal(t, "349000.30", c.Groups[0].Subtotal.StringFixed(2))
	assert.Equal(t, "0.00", c.Groups[1].Subtotal.StringFixed(2))
	assert.Equal(t, "349000.30", c.Subtotal.StringFixed(2), "lines with issues are not counted")
	assert.Equal(t, "118001.00", c.Groups[1].Lines[0].LineTotal.StringFixed(2), "line totals are still shown")
	assert.Equal(t, int32(7), c.ItemCount, "badge counts every unit in the cart")
	assert.True(t, c.HasIssues)
}

func TestBuildCart_Empty(t *testing.T) {
	c := buildCart(nil)
	assert.NotNil(t, c.Groups, "JSON must show [] not null")
	assert.Empty(t, c.Groups)
	assert.Equal(t, int32(0), c.ItemCount)
	assert.True(t, c.Subtotal.IsZero())
	assert.False(t, c.HasIssues)
}
