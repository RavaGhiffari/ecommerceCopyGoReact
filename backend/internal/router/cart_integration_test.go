//go:build integration

// End-to-end tests of Phase 4 (cart) against the real marketplace_test DB.
package router_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cartFixture: two sellers with one product each, and a buyer.
type cartFixture struct {
	app                *testApp
	sellerA, sellerB   string // sessions
	buyer              string
	productA, productB string // product ids
	variantA, variantB string // variant ids
	variantAPath       string // seller URL of variant A (for stock/price changes)
}

func newCartFixture(t *testing.T) cartFixture {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	f := cartFixture{app: app}
	f.sellerA = app.newSeller("Toko Alpha")
	f.sellerB = app.newSeller("Toko Beta")
	f.buyer = app.register("Pembeli", "pembeli@example.com", "rahasia-123")

	pa := app.createProduct(f.sellerA, productBody("Earbuds", c.audio, variantBody{"Hitam", "AL-1", "100000.50", 5}))
	pb := app.createProduct(f.sellerB, productBody("Mouse", c.komputer, variantBody{"Hitam", "BE-1", "50000", 200}))
	f.productA, f.productB = pa["id"].(string), pb["id"].(string)
	f.variantA = pa["variants"].([]any)[0].(map[string]any)["id"].(string)
	f.variantB = pb["variants"].([]any)[0].(map[string]any)["id"].(string)
	f.variantAPath = fmt.Sprintf("/api/v1/seller/products/%s/variants/%s", f.productA, f.variantA)
	return f
}

func (f cartFixture) add(session, variantID string, qty int) result {
	f.app.t.Helper()
	return f.app.do("POST", "/api/v1/cart/items", fmt.Sprintf(`{"variantId":%q,"quantity":%d}`, variantID, qty), session)
}

func cartStores(r result) []any { s, _ := r.data()["stores"].([]any); return s }

func firstItem(r result) map[string]any {
	return cartStores(r)[0].(map[string]any)["items"].([]any)[0].(map[string]any)
}

func TestCart_RequiresLogin(t *testing.T) {
	f := newCartFixture(t)
	assert.Equal(t, http.StatusUnauthorized, f.app.do("GET", "/api/v1/cart", "", "").status)
	assert.Equal(t, http.StatusUnauthorized, f.add("", f.variantA, 1).status)
}

func TestCart_EmptyCartIsNotAnError(t *testing.T) {
	f := newCartFixture(t)
	res := f.app.do("GET", "/api/v1/cart", "", f.buyer)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Empty(t, cartStores(res))
	assert.Equal(t, "0.00", res.data()["subtotal"])
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM carts"), "reading does not create a cart")
}

func TestCart_AddIncrementsAndGroupsByStore(t *testing.T) {
	f := newCartFixture(t)

	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 2).status)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)
	res := f.add(f.buyer, f.variantB, 4)
	require.Equal(t, http.StatusOK, res.status, res.raw)

	stores := cartStores(res)
	require.Len(t, stores, 2, "one group per store")
	alpha := stores[0].(map[string]any)
	assert.Equal(t, "Toko Alpha", alpha["store"].(map[string]any)["name"])
	items := alpha["items"].([]any)
	require.Len(t, items, 1, "adding the same variant again increases the quantity")
	assert.Equal(t, float64(3), items[0].(map[string]any)["quantity"])
	assert.Equal(t, "300001.50", items[0].(map[string]any)["lineTotal"])
	assert.Equal(t, "300001.50", alpha["subtotal"])

	assert.Equal(t, "500001.50", res.data()["subtotal"], "3 × 100000.50 + 4 × 50000, exact decimals")
	assert.Equal(t, float64(7), res.data()["itemCount"])
	assert.Equal(t, 2, f.app.count("SELECT count(*) FROM cart_items"))
}

func TestCart_IsNotAReservation(t *testing.T) {
	f := newCartFixture(t)
	before := f.app.count("SELECT count(*) FROM inventory_movements")

	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 5).status)

	assert.Equal(t, 5, f.app.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", f.variantA), "stock unchanged")
	assert.Equal(t, 0, f.app.count("SELECT reserved_quantity FROM inventory WHERE variant_id = $1", f.variantA), "nothing reserved")
	assert.Equal(t, before, f.app.count("SELECT count(*) FROM inventory_movements"), "no ledger entry")
}

func TestCart_QuantityRules(t *testing.T) {
	f := newCartFixture(t)

	assert.Equal(t, http.StatusUnprocessableEntity, f.add(f.buyer, f.variantA, 0).status)
	assert.Equal(t, http.StatusUnprocessableEntity, f.add(f.buyer, f.variantA, 100).status)

	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 4).status)
	res := f.add(f.buyer, f.variantA, 2) // 4 + 2 > stock 5
	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "QUANTITY_EXCEEDS_STOCK", res.errorCode())
	details := res.body["error"].(map[string]any)["details"].(map[string]any)
	assert.Equal(t, float64(5), details["available"])
	assert.Equal(t, float64(4), details["inCart"])

	// Stock 200, so the 99-per-line cap is what stops this one.
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 98).status)
	res = f.add(f.buyer, f.variantB, 2)
	assert.Equal(t, http.StatusUnprocessableEntity, res.status, res.raw)

	// Out of stock.
	require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":0}`, f.sellerA).status)
	other := f.app.register("Lain", "lain@example.com", "rahasia-123")
	assert.Equal(t, "OUT_OF_STOCK", f.add(other, f.variantA, 1).errorCode())

	assert.Equal(t, http.StatusNotFound, f.add(f.buyer, "00000000-0000-0000-0000-000000000000", 1).status)
}

func TestCart_SellerCannotBuyOwnProduct(t *testing.T) {
	f := newCartFixture(t)
	res := f.add(f.sellerA, f.variantA, 1)
	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "OWN_PRODUCT", res.errorCode())
	assert.Equal(t, http.StatusOK, f.add(f.sellerA, f.variantB, 1).status, "but sellers can buy from other stores")
}

func TestCart_ArchivedProductRejectedAndFlagged(t *testing.T) {
	f := newCartFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 2).status)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 1).status)

	require.Equal(t, http.StatusNoContent, f.app.do("DELETE", "/api/v1/seller/products/"+f.productA, "", f.sellerA).status)

	res := f.app.do("GET", "/api/v1/cart", "", f.buyer)
	require.Equal(t, http.StatusOK, res.status, "a broken line never breaks the cart")
	line := firstItem(res)
	assert.Equal(t, "unavailable", line["issue"], "stays in the cart, flagged")
	assert.Equal(t, true, res.data()["hasIssues"])
	assert.Equal(t, "50000.00", res.data()["subtotal"], "flagged lines are not counted")

	other := f.app.register("Lain", "lain@example.com", "rahasia-123")
	assert.Equal(t, "VARIANT_UNAVAILABLE", f.add(other, f.variantA, 1).errorCode())
	assert.Equal(t, "VARIANT_UNAVAILABLE",
		f.app.do("PATCH", "/api/v1/cart/items/"+line["id"].(string), `{"quantity":1}`, f.buyer).errorCode())
	assert.Equal(t, http.StatusOK, f.app.do("DELETE", "/api/v1/cart/items/"+line["id"].(string), "", f.buyer).status,
		"it can still be removed")
}

func TestCart_StockDropFlagsInsufficientStock(t *testing.T) {
	f := newCartFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 4).status)
	require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":2}`, f.sellerA).status)

	line := firstItem(f.app.do("GET", "/api/v1/cart", "", f.buyer))
	assert.Equal(t, "insufficient_stock", line["issue"])
	assert.Equal(t, float64(2), line["availableQuantity"])

	id := line["id"].(string)
	assert.Equal(t, "QUANTITY_EXCEEDS_STOCK", f.app.do("PATCH", "/api/v1/cart/items/"+id, `{"quantity":3}`, f.buyer).errorCode())
	fixed := f.app.do("PATCH", "/api/v1/cart/items/"+id, `{"quantity":2}`, f.buyer)
	require.Equal(t, http.StatusOK, fixed.status, fixed.raw)
	assert.Nil(t, firstItem(fixed)["issue"])
}

func TestCart_PriceIsAlwaysCurrent(t *testing.T) {
	f := newCartFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 2).status)

	require.Equal(t, http.StatusOK, f.app.do("PATCH", f.variantAPath, `{"price":"90000"}`, f.sellerA).status)

	line := firstItem(f.app.do("GET", "/api/v1/cart", "", f.buyer))
	assert.Equal(t, "90000.00", line["unitPrice"], "cart shows the current price; orders will snapshot it")
	assert.Equal(t, "180000.00", line["lineTotal"])
}

func TestCart_OtherUsersCannotTouchMyItems(t *testing.T) {
	f := newCartFixture(t)
	res := f.add(f.buyer, f.variantA, 2)
	id := firstItem(res)["id"].(string)
	intruder := f.app.register("Penyusup", "penyusup@example.com", "rahasia-123")
	f.add(intruder, f.variantB, 1) // the intruder has a cart of their own

	assert.Equal(t, http.StatusNotFound, f.app.do("PATCH", "/api/v1/cart/items/"+id, `{"quantity":1}`, intruder).status)
	assert.Equal(t, http.StatusNotFound, f.app.do("DELETE", "/api/v1/cart/items/"+id, "", intruder).status)

	mine := firstItem(f.app.do("GET", "/api/v1/cart", "", f.buyer))
	assert.Equal(t, float64(2), mine["quantity"], "unchanged")
	assert.Len(t, cartStores(f.app.do("GET", "/api/v1/cart", "", intruder)), 1, "the intruder only sees their own cart")
}

func TestCart_UpdateRemoveAndPersistAcrossSessions(t *testing.T) {
	f := newCartFixture(t)
	id := firstItem(f.add(f.buyer, f.variantB, 1))["id"].(string)

	upd := f.app.do("PATCH", "/api/v1/cart/items/"+id, `{"quantity":7}`, f.buyer)
	require.Equal(t, http.StatusOK, upd.status, upd.raw)
	assert.Equal(t, float64(7), firstItem(upd)["quantity"])

	// Log out and back in: the cart is stored server-side, not in the session.
	require.Equal(t, http.StatusNoContent, f.app.do("POST", "/api/v1/auth/logout", "", f.buyer).status)
	again := f.app.login("pembeli@example.com", "rahasia-123")
	assert.Equal(t, float64(7), firstItem(f.app.do("GET", "/api/v1/cart", "", again))["quantity"])

	del := f.app.do("DELETE", "/api/v1/cart/items/"+id, "", again)
	require.Equal(t, http.StatusOK, del.status)
	assert.Empty(t, cartStores(del))
	assert.Equal(t, http.StatusNotFound, f.app.do("DELETE", "/api/v1/cart/items/"+id, "", again).status, "already gone")
}

func TestCart_ConcurrentAddsOfTheSameVariant(t *testing.T) {
	f := newCartFixture(t)

	// 10 simultaneous "add 1" clicks. The cart row lock serialises them, so
	// the result is one line with quantity 10 (no lost update, no duplicate row).
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.add(f.buyer, f.variantB, 1)
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM carts"), "one cart even if the first adds raced")
	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM cart_items"))
	assert.Equal(t, 10, f.app.count("SELECT quantity FROM cart_items"))
}
