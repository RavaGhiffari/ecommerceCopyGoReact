package checkout

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/shipping"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

var (
	buyer    = uuid.New()
	bandung  = StoreRef{ID: uuid.New(), Name: "Toko Bandung", Location: "Bandung"}
	jakarta  = StoreRef{ID: uuid.New(), Name: "Toko Jakarta", Location: "Jakarta Selatan"}
	toBdg    = addresses.Address{City: "bandung"}
	dec      = decimal.RequireFromString
	defRates = shipping.DefaultRates()
)

func row(store StoreRef, price string, qty, available int32) lineRow {
	return lineRow{
		ItemID: uuid.New(), VariantID: uuid.New(), ProductID: uuid.New(), Quantity: qty, Price: dec(price),
		ProductStatus: "active", Store: store, StoreStatus: "active", StoreOwnerID: uuid.New(),
		AvailableQuantity: available,
	}
}

func TestBuildQuote_TwoStoresExactTotals(t *testing.T) {
	rows := []lineRow{
		row(bandung, "100000.50", 3, 10),
		row(bandung, "0.10", 1, 10),
		row(jakarta, "50000", 2, 10),
	}
	choice := map[uuid.UUID]shipping.Method{bandung.ID: shipping.SameDay, jakarta.ID: shipping.Express}

	q, err := buildQuote(rows, buyer, toBdg, choice, defRates, true)
	require.NoError(t, err)
	require.Len(t, q.Groups, 2)

	b := q.Groups[0]
	assert.Equal(t, "300001.60", b.Subtotal.StringFixed(2), "3 × 100000.50 + 0.10, no float drift")
	assert.Equal(t, "45000.00", b.ShippingFee.StringFixed(2))
	assert.Equal(t, "345001.60", b.Total.StringFixed(2))
	assert.Len(t, b.Options, 3, "same city: Same Day offered")
	assert.Len(t, q.Groups[1].Options, 2, "other city: no Same Day")

	assert.Equal(t, "400001.60", q.ItemsTotal.StringFixed(2))
	assert.Equal(t, "75000.00", q.ShippingTotal.StringFixed(2))
	assert.Equal(t, "475001.60", q.GrandTotal.StringFixed(2))
	assert.False(t, q.HasIssues)
}

func TestBuildQuote_QuoteDefaultsToRegularButPlacingNeedsAChoice(t *testing.T) {
	rows := []lineRow{row(jakarta, "1000", 1, 5)}

	q, err := buildQuote(rows, buyer, toBdg, nil, defRates, false)
	require.NoError(t, err)
	assert.Equal(t, shipping.Regular, q.Groups[0].Method)
	assert.Equal(t, "16000.00", q.GrandTotal.StringFixed(2))

	_, err = buildQuote(rows, buyer, toBdg, nil, defRates, true)
	var fields validation.Errors
	require.True(t, errors.As(err, &fields))
	assert.Equal(t, "is required", fields["shipping."+jakarta.ID.String()])
}

func TestBuildQuote_SameDayToAnotherCityIsRejected(t *testing.T) {
	rows := []lineRow{row(jakarta, "1000", 1, 5)}
	_, err := buildQuote(rows, buyer, toBdg, map[uuid.UUID]shipping.Method{jakarta.ID: shipping.SameDay}, defRates, false)
	var na *MethodNotAllowedError
	require.True(t, errors.As(err, &na))
	assert.Contains(t, na.Fields, "shipping."+jakarta.ID.String())
}

func TestBuildQuote_ShippingForAStoreNotInTheCart(t *testing.T) {
	rows := []lineRow{row(jakarta, "1000", 1, 5)}
	stranger := uuid.New()
	_, err := buildQuote(rows, buyer, toBdg, map[uuid.UUID]shipping.Method{jakarta.ID: shipping.Regular, stranger: shipping.Regular}, defRates, true)
	var fields validation.Errors
	require.True(t, errors.As(err, &fields))
	assert.Equal(t, "is not a store in your cart", fields["shipping."+stranger.String()])
}

func TestBuildQuote_IssuesAreFlaggedAndLeftOutOfTotals(t *testing.T) {
	archived := row(bandung, "10", 1, 5)
	archived.ProductStatus = "archived"
	suspended := row(bandung, "10", 1, 5)
	suspended.StoreStatus = "suspended"
	own := row(bandung, "10", 1, 5)
	own.StoreOwnerID = buyer
	short := row(bandung, "10", 4, 3)
	empty := row(bandung, "10", 1, 0)
	fine := row(bandung, "7", 2, 5)

	q, err := buildQuote([]lineRow{archived, suspended, own, short, empty, fine}, buyer, toBdg, nil, defRates, false)
	require.NoError(t, err)
	assert.True(t, q.HasIssues)

	got := map[uuid.UUID]string{}
	for _, l := range q.Groups[0].Lines {
		got[l.ItemID] = l.Issue
	}
	assert.Equal(t, IssueUnavailable, got[archived.ItemID])
	assert.Equal(t, IssueUnavailable, got[suspended.ItemID])
	assert.Equal(t, IssueOwnProduct, got[own.ItemID])
	assert.Equal(t, IssueInsufficientStock, got[short.ItemID])
	assert.Equal(t, IssueOutOfStock, got[empty.ItemID])
	assert.Equal(t, "", got[fine.ItemID])
	assert.Len(t, q.IssueLines(), 5)
	assert.Equal(t, "14.00", q.ItemsTotal.StringFixed(2), "only the fine line counts")
}

func TestInputHash_IgnoresMapOrderButNotContent(t *testing.T) {
	addr, s1, s2 := uuid.New(), uuid.New(), uuid.New()
	a := Input{AddressID: addr, Shipping: map[uuid.UUID]shipping.Method{s1: shipping.Regular, s2: shipping.Express}}
	b := Input{AddressID: addr, Shipping: map[uuid.UUID]shipping.Method{s2: shipping.Express, s1: shipping.Regular}}
	assert.Equal(t, a.Hash(), b.Hash())
	assert.Len(t, a.Hash(), 32)

	c := Input{AddressID: addr, Shipping: map[uuid.UUID]shipping.Method{s1: shipping.Express, s2: shipping.Express}}
	assert.NotEqual(t, a.Hash(), c.Hash(), "another method is another request")
	d := Input{AddressID: uuid.New(), Shipping: a.Shipping}
	assert.NotEqual(t, a.Hash(), d.Hash(), "another address is another request")
}

func TestParseInput(t *testing.T) {
	_, err := ParseInput(nil, map[string]string{"not-a-uuid": "regular", uuid.NewString(): "drone"})
	var fields validation.Errors
	require.True(t, errors.As(err, &fields))
	assert.Equal(t, "is required", fields["addressId"])
	assert.Equal(t, "is not a store in your cart", fields["shipping.not-a-uuid"])
	assert.Len(t, fields, 3)

	addr, store := uuid.New(), uuid.New()
	in, err := ParseInput(&addr, map[string]string{store.String(): "same_day"})
	require.NoError(t, err)
	assert.Equal(t, shipping.SameDay, in.Shipping[store])
}
