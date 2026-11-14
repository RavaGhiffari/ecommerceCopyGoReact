//go:build integration

// End-to-end tests of Phase 5b (checkout core) against marketplace_test.
package router_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkoutFixture: two Bandung stores with one product each, and a buyer
// with a Bandung address.
type checkoutFixture struct {
	cartFixture
	storeA, storeB string // store ids
	address        string // buyer's address id (Bandung)
}

func newCheckoutFixture(t *testing.T) checkoutFixture {
	f := checkoutFixture{cartFixture: newCartFixture(t)}
	f.storeA = f.app.do("GET", "/api/v1/seller/store", "", f.sellerA).data()["id"].(string)
	f.storeB = f.app.do("GET", "/api/v1/seller/store", "", f.sellerB).data()["id"].(string)
	f.address = f.app.addAddress(f.buyer, "Rumah", false)["id"].(string)
	return f
}

func checkoutBody(t *testing.T, addressID string, shipping map[string]string) string {
	return toJSON(t, map[string]any{"addressId": addressID, "shipping": shipping})
}

func (f checkoutFixture) place(session, key, body string) result {
	f.app.t.Helper()
	return f.app.do("POST", "/api/v1/checkout", body, session, "Idempotency-Key", key)
}

// placeAll checks out the buyer's cart with regular shipping for every store.
func (f checkoutFixture) placeAll(session, addressID string, stores ...string) result {
	ship := map[string]string{}
	for _, s := range stores {
		ship[s] = "regular"
	}
	return f.place(session, uuid.NewString(), checkoutBody(f.app.t, addressID, ship))
}

// ledgerMismatches counts variants whose quantities differ from the sum of
// their ledger rows. The invariant: it is always 0.
func (a *testApp) ledgerMismatches() int {
	return a.count(`
		SELECT count(*) FROM inventory i
		WHERE i.available_quantity <> (SELECT COALESCE(SUM(available_delta), 0) FROM inventory_movements m WHERE m.variant_id = i.variant_id)
		   OR i.reserved_quantity  <> (SELECT COALESCE(SUM(reserved_delta), 0)  FROM inventory_movements m WHERE m.variant_id = i.variant_id)`)
}

// writes counts every row a checkout could have written.
func (a *testApp) writes() map[string]int {
	out := map[string]int{}
	for _, table := range []string{"checkouts", "orders", "order_items", "order_status_histories", "payments", "inventory_movements"} {
		out[table] = a.count("SELECT count(*) FROM " + table)
	}
	return out
}

var orderNumber = regexp.MustCompile(`^PK-[0-9]{8}-[0-9A-HJKMNP-TV-Z]{6}$`)

func TestCheckout_RequestRules(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)
	body := checkoutBody(t, f.address, map[string]string{f.storeA: "regular"})

	assert.Equal(t, http.StatusUnauthorized, f.place("", uuid.NewString(), body).status)

	res := f.app.do("POST", "/api/v1/checkout", body, f.buyer)
	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, "IDEMPOTENCY_KEY_REQUIRED", res.errorCode())
	assert.Equal(t, "IDEMPOTENCY_KEY_REQUIRED", f.place(f.buyer, "not-a-uuid", body).errorCode())

	res = f.place(f.buyer, uuid.NewString(), `{"shipping":{"x":"drone"}}`)
	assert.Equal(t, http.StatusUnprocessableEntity, res.status, res.raw)
	assert.Contains(t, res.raw, "addressId")

	res = f.place(f.buyer, uuid.NewString(), `{"addressId":"`+f.address+`","total":"1.00"}`)
	assert.Equal(t, http.StatusBadRequest, res.status, "unknown fields such as a client total are rejected")

	res = f.place(f.buyer, uuid.NewString(), checkoutBody(t, f.address, nil))
	assert.Equal(t, http.StatusUnprocessableEntity, res.status, "placing needs a method for every store")
	assert.Contains(t, res.raw, "shipping."+f.storeA)

	res = f.place(f.buyer, uuid.NewString(), checkoutBody(t, f.address, map[string]string{f.storeA: "regular", f.storeB: "regular"}))
	assert.Equal(t, http.StatusUnprocessableEntity, res.status, "store B is not in the cart")

	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM checkouts"))
}

func TestCheckout_QuoteIsReadOnly(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 2).status)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 1).status)
	before := f.app.writes()

	res := f.app.do("POST", "/api/v1/checkout/quote", checkoutBody(t, f.address, map[string]string{f.storeA: "same_day"}), f.buyer)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	d := res.data()
	stores := d["stores"].([]any)
	require.Len(t, stores, 2)
	alpha := stores[0].(map[string]any)
	assert.Equal(t, "same_day", alpha["shippingMethod"])
	assert.Len(t, alpha["shippingOptions"], 3, "store and address both in Bandung")
	assert.Equal(t, "regular", stores[1].(map[string]any)["shippingMethod"], "a quote defaults to regular")
	// 2 × 100000.50 + 50000 = 250001.00; 45000 + 15000 shipping.
	assert.Equal(t, "250001.00", d["itemsTotal"])
	assert.Equal(t, "60000.00", d["shippingTotal"])
	assert.Equal(t, "310001.00", d["grandTotal"])
	assert.Equal(t, false, d["hasIssues"])

	assert.Equal(t, before, f.app.writes(), "a quote writes nothing")
}

func TestCheckout_HappyPathTwoStores(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 2).status) // stock 5
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 3).status) // stock 200

	res := f.place(f.buyer, uuid.NewString(), checkoutBody(t, f.address, map[string]string{f.storeA: "same_day", f.storeB: "express"}))
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	d := res.data()
	checkoutID := d["id"].(string)

	// Totals computed by the server: 2 × 100000.50 + 3 × 50000 + 45000 + 30000.
	assert.Equal(t, "pending_payment", d["status"])
	assert.Equal(t, "350001.00", d["itemsTotal"])
	assert.Equal(t, "75000.00", d["shippingTotal"])
	assert.Equal(t, "425001.00", d["grandTotal"])

	pay := d["payment"].(map[string]any)
	assert.Equal(t, "pending", pay["status"])
	assert.Equal(t, "425001.00", pay["amount"])
	assert.Regexp(t, `^SIM-[0-9A-Z]{16}$`, pay["reference"])
	assert.Equal(t, d["expiresAt"], pay["expiresAt"])

	ords := d["orders"].([]any)
	require.Len(t, ords, 2, "one order per store")
	alpha := ords[0].(map[string]any)
	assert.Regexp(t, orderNumber, alpha["orderNumber"])
	assert.Equal(t, "Toko Alpha", alpha["store"].(map[string]any)["name"])
	assert.Equal(t, "same_day", alpha["shippingMethod"])
	assert.Equal(t, "200001.00", alpha["itemsSubtotal"])
	assert.Equal(t, "245001.00", alpha["total"])
	assert.Equal(t, "Bandung", alpha["shippingAddress"].(map[string]any)["city"])
	item := alpha["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "Earbuds", item["productName"])
	assert.Equal(t, "AL-1", item["sku"])
	assert.Equal(t, "100000.50", item["unitPrice"])

	// Stock moved from available to reserved, with one ledger row per line.
	assert.Equal(t, 3, f.app.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", f.variantA))
	assert.Equal(t, 2, f.app.count("SELECT reserved_quantity FROM inventory WHERE variant_id = $1", f.variantA))
	assert.Equal(t, 197, f.app.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", f.variantB))
	assert.Equal(t, 2, f.app.count(
		"SELECT count(*) FROM inventory_movements WHERE movement_type = 'reservation' AND reference_type = 'checkout' AND reference_id = $1", checkoutID))
	assert.Equal(t, 0, f.app.ledgerMismatches())

	// History, and the cart is empty.
	assert.Equal(t, 2, f.app.count("SELECT count(*) FROM order_status_histories WHERE from_status IS NULL AND to_status = 'pending_payment'"))
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM cart_items"))
	assert.Equal(t, float64(0), f.app.do("GET", "/api/v1/cart", "", f.buyer).data()["itemCount"])

	// The same data through the read endpoints.
	got := f.app.do("GET", "/api/v1/checkout/"+checkoutID, "", f.buyer)
	require.Equal(t, http.StatusOK, got.status, got.raw)
	assert.Equal(t, d["grandTotal"], got.data()["grandTotal"])

	list := f.app.do("GET", "/api/v1/orders", "", f.buyer)
	require.Equal(t, http.StatusOK, list.status, list.raw)
	assert.Equal(t, float64(2), list.data()["total"])
	assert.Len(t, listItems(list), 2)

	one := f.app.do("GET", "/api/v1/orders/"+alpha["id"].(string), "", f.buyer)
	require.Equal(t, http.StatusOK, one.status, one.raw)
	hist := one.data()["history"].([]any)
	require.Len(t, hist, 1)
	assert.Nil(t, hist[0].(map[string]any)["fromStatus"])
	assert.Equal(t, "pending_payment", hist[0].(map[string]any)["toStatus"])
}

func TestCheckout_OrdersKeepTheirSnapshots(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)
	res := f.placeAll(f.buyer, f.address, f.storeA)
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	orderID := res.data()["orders"].([]any)[0].(map[string]any)["id"].(string)

	// Later changes to the product, the store and the address...
	ctx := context.Background()
	_, err := f.app.pool.Exec(ctx, "UPDATE product_variants SET price = 1, name = 'Baru', sku = 'AL-NEW' WHERE id = $1", f.variantA)
	require.NoError(t, err)
	_, err = f.app.pool.Exec(ctx, "UPDATE products SET name = 'Nama Baru' WHERE id = $1", f.productA)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, f.app.do("PATCH", "/api/v1/seller/store", `{"name":"Toko Ganti Nama"}`, f.sellerA).status)
	require.Equal(t, http.StatusNoContent, f.app.do("DELETE", "/api/v1/me/addresses/"+f.address, "", f.buyer).status)

	// ...do not change the order.
	o := f.app.do("GET", "/api/v1/orders/"+orderID, "", f.buyer).data()
	assert.Equal(t, "Toko Alpha", o["store"].(map[string]any)["name"])
	assert.Equal(t, "Jl. Melati No. 5", o["shippingAddress"].(map[string]any)["addressLine"])
	item := o["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "Earbuds", item["productName"])
	assert.Equal(t, "Hitam", item["variantName"])
	assert.Equal(t, "AL-1", item["sku"])
	assert.Equal(t, "100000.50", item["unitPrice"])
	assert.Equal(t, "115000.50", o["total"])
}

func TestCheckout_IdempotencyKey(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 2).status)
	key := uuid.NewString()
	body := checkoutBody(t, f.address, map[string]string{f.storeA: "regular"})

	first := f.place(f.buyer, key, body)
	require.Equal(t, http.StatusCreated, first.status, first.raw)
	after := f.app.writes()

	// A retry (lost response, double click): same checkout, nothing new.
	// The cart is empty by now, which proves the key is checked first.
	again := f.place(f.buyer, key, body)
	require.Equal(t, http.StatusOK, again.status, again.raw)
	assert.Equal(t, first.data()["id"], again.data()["id"])
	assert.Equal(t, after, f.app.writes(), "no second checkout, order or reservation")
	assert.Equal(t, 3, f.app.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", f.variantA), "reserved once")

	// The same key for a different request is a client bug: 409.
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 1).status)
	other := f.place(f.buyer, key, checkoutBody(t, f.address, map[string]string{f.storeB: "express"}))
	assert.Equal(t, http.StatusConflict, other.status, other.raw)
	assert.Equal(t, "IDEMPOTENCY_KEY_REUSED", other.errorCode())

	// Keys are per user: another buyer may use the same UUID.
	buyer2 := f.app.register("Pembeli Dua", "dua@example.com", "rahasia-123")
	addr2 := f.app.addAddress(buyer2, "Rumah", false)["id"].(string)
	require.Equal(t, http.StatusOK, f.add(buyer2, f.variantB, 1).status)
	res := f.place(buyer2, key, checkoutBody(t, addr2, map[string]string{f.storeB: "regular"}))
	assert.Equal(t, http.StatusCreated, res.status, res.raw)
}

func TestCheckout_ConcurrentSubmitsWithTheSameKey(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)
	key := uuid.NewString()
	body := checkoutBody(t, f.address, map[string]string{f.storeA: "regular"})

	statuses := map[int]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := f.place(f.buyer, key, body)
			mu.Lock()
			statuses[res.status]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Equal(t, map[int]int{http.StatusCreated: 1, http.StatusOK: 4}, statuses)
	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM checkouts"))
	assert.Equal(t, 1, f.app.count("SELECT reserved_quantity FROM inventory WHERE variant_id = $1", f.variantA))
}

func TestCheckout_StockChangedRollsEverythingBack(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 3).status)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 1).status)
	// The seller lowers the stock after the buyer filled the cart.
	require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":2}`, f.sellerA).status)
	before := f.app.writes()

	res := f.placeAll(f.buyer, f.address, f.storeA, f.storeB)
	require.Equal(t, http.StatusConflict, res.status, res.raw)
	assert.Equal(t, "CART_HAS_ISSUES", res.errorCode())
	items := res.body["error"].(map[string]any)["details"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1, "only the short line is reported")
	assert.Equal(t, "insufficient_stock", items[0].(map[string]any)["issue"])
	assert.Equal(t, float64(2), items[0].(map[string]any)["availableQuantity"])

	// Rollback proof: not one row written, stock and cart untouched.
	assert.Equal(t, before, f.app.writes())
	assert.Equal(t, 2, f.app.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", f.variantA))
	assert.Equal(t, 200, f.app.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", f.variantB), "store B line was fine but is not reserved either")
	assert.Equal(t, 0, f.app.count("SELECT COALESCE(SUM(reserved_quantity), 0) FROM inventory"))
	assert.Equal(t, 2, f.app.count("SELECT count(*) FROM cart_items"))
}

func TestCheckout_UnbuyableLinesBlockTheCheckout(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 1).status)

	// Archived product.
	require.Equal(t, http.StatusNoContent, f.app.do("DELETE", "/api/v1/seller/products/"+f.productA, "", f.sellerA).status)
	res := f.placeAll(f.buyer, f.address, f.storeA, f.storeB)
	assert.Equal(t, "CART_HAS_ISSUES", res.errorCode(), res.raw)
	assert.Contains(t, res.raw, `"issue":"unavailable"`)

	// Suspended store (admin moderation; set directly here).
	require.Equal(t, http.StatusOK, f.app.do("DELETE", "/api/v1/cart/items/"+f.cartItemID(f.buyer, f.variantA), "", f.buyer).status)
	_, err := f.app.pool.Exec(context.Background(), "UPDATE stores SET status = 'suspended' WHERE id = $1", f.storeB)
	require.NoError(t, err)
	res = f.placeAll(f.buyer, f.address, f.storeB)
	assert.Equal(t, "CART_HAS_ISSUES", res.errorCode(), res.raw)

	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM checkouts"))
}

func TestCheckout_CannotBuyFromOwnStore(t *testing.T) {
	f := newCheckoutFixture(t)
	// The cart API refuses own products, but a buyer can open a store after
	// adding items. Simulate that by putting seller A's own variant in their cart.
	require.Equal(t, http.StatusOK, f.add(f.sellerA, f.variantB, 1).status)
	_, err := f.app.pool.Exec(context.Background(), `
		INSERT INTO cart_items (cart_id, variant_id, quantity)
		SELECT c.id, $1, 1 FROM carts c JOIN stores s ON s.owner_id = c.user_id WHERE s.id = $2`, f.variantA, f.storeA)
	require.NoError(t, err)
	addr := f.app.addAddress(f.sellerA, "Toko", false)["id"].(string)

	res := f.placeAll(f.sellerA, addr, f.storeA, f.storeB)
	assert.Equal(t, http.StatusConflict, res.status, res.raw)
	assert.Contains(t, res.raw, `"issue":"own_product"`)
}

func TestCheckout_AddressCartAndShippingErrors(t *testing.T) {
	f := newCheckoutFixture(t)

	res := f.placeAll(f.buyer, f.address)
	assert.Equal(t, http.StatusUnprocessableEntity, res.status, res.raw)
	assert.Equal(t, "CART_EMPTY", res.errorCode())

	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)

	other := f.app.register("Lain", "lain@example.com", "rahasia-123")
	foreign := f.app.addAddress(other, "Rumah Orang", false)["id"].(string)
	res = f.placeAll(f.buyer, foreign, f.storeA)
	assert.Equal(t, http.StatusNotFound, res.status, "someone else's address")
	assert.Equal(t, "ADDRESS_NOT_FOUND", res.errorCode())

	// Same Day from Bandung to Jakarta is not offered.
	jkt := f.app.do("POST", "/api/v1/me/addresses", addressBody("Kantor", "city", "Jakarta Selatan", "postalCode", "12160"), f.buyer)
	require.Equal(t, http.StatusCreated, jkt.status, jkt.raw)
	res = f.place(f.buyer, uuid.NewString(), checkoutBody(t, jkt.data()["id"].(string), map[string]string{f.storeA: "same_day"}))
	assert.Equal(t, http.StatusUnprocessableEntity, res.status, res.raw)
	assert.Equal(t, "SHIPPING_METHOD_NOT_ALLOWED", res.errorCode())

	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM checkouts"))
}

func TestCheckout_OtherBuyersSeeNothing(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)
	res := f.placeAll(f.buyer, f.address, f.storeA)
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	checkoutID := res.data()["id"].(string)
	orderID := res.data()["orders"].([]any)[0].(map[string]any)["id"].(string)

	other := f.app.register("Lain", "lain@example.com", "rahasia-123")
	assert.Equal(t, http.StatusNotFound, f.app.do("GET", "/api/v1/checkout/"+checkoutID, "", other).status)
	assert.Equal(t, http.StatusNotFound, f.app.do("GET", "/api/v1/orders/"+orderID, "", other).status)
	assert.Equal(t, "ORDER_NOT_FOUND", f.app.do("GET", "/api/v1/orders/not-a-uuid", "", other).errorCode())
	assert.Empty(t, listItems(f.app.do("GET", "/api/v1/orders", "", other)))
	// The seller is not the buyer either (seller views arrive in 5d).
	assert.Equal(t, http.StatusNotFound, f.app.do("GET", "/api/v1/orders/"+orderID, "", f.sellerA).status)

	assert.Equal(t, http.StatusUnprocessableEntity, f.app.do("GET", "/api/v1/orders?pageSize=500", "", f.buyer).status)
}

// The mandatory overselling test: 10 buyers, 3 units, all at once.
func TestCheckout_ConcurrentBuyersNeverOversell(t *testing.T) {
	f := newCheckoutFixture(t)
	require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":3}`, f.sellerA).status)

	type buyer struct{ session, address string }
	buyers := make([]buyer, 10)
	for i := range buyers {
		s := f.app.register(fmt.Sprintf("Pembeli %d", i), fmt.Sprintf("pembeli%d@example.com", i), "rahasia-123")
		buyers[i] = buyer{s, f.app.addAddress(s, "Rumah", false)["id"].(string)}
		require.Equal(t, http.StatusOK, f.add(s, f.variantA, 1).status, "the cart does not reserve, so all 10 fit")
	}

	statuses := map[int]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, b := range buyers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res := f.placeAll(b.session, b.address, f.storeA)
			mu.Lock()
			statuses[res.status]++
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, map[int]int{http.StatusCreated: 3, http.StatusConflict: 7}, statuses)
	assert.Equal(t, 0, f.app.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", f.variantA))
	assert.Equal(t, 3, f.app.count("SELECT reserved_quantity FROM inventory WHERE variant_id = $1", f.variantA))
	assert.Equal(t, 3, f.app.count("SELECT count(*) FROM checkouts"))
	assert.Equal(t, 0, f.app.ledgerMismatches())
}

// Two buyers whose carts hold the same two variants, added in opposite
// order. Without a fixed lock order each could hold one row and wait for the
// other's (PostgreSQL would then abort one with a deadlock error -> 500).
func TestCheckout_SharedVariantsDoNotDeadlock(t *testing.T) {
	f := newCheckoutFixture(t)
	b1, b2 := f.buyer, f.app.register("Pembeli Dua", "dua@example.com", "rahasia-123")
	a1, a2 := f.address, f.app.addAddress(b2, "Rumah", false)["id"].(string)

	for round := range 5 {
		require.Equal(t, http.StatusOK, f.add(b1, f.variantA, 1).status)
		require.Equal(t, http.StatusOK, f.add(b1, f.variantB, 1).status)
		require.Equal(t, http.StatusOK, f.add(b2, f.variantB, 1).status)
		require.Equal(t, http.StatusOK, f.add(b2, f.variantA, 1).status)
		// Keep enough stock for every round.
		require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":50}`, f.sellerA).status)

		var r1, r2 result
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); r1 = f.placeAll(b1, a1, f.storeA, f.storeB) }()
		go func() { defer wg.Done(); r2 = f.placeAll(b2, a2, f.storeA, f.storeB) }()
		wg.Wait()
		require.Equal(t, http.StatusCreated, r1.status, "round %d: %s", round, r1.raw)
		require.Equal(t, http.StatusCreated, r2.status, "round %d: %s", round, r2.raw)
	}
	assert.Equal(t, 10, f.app.count("SELECT count(*) FROM checkouts"))
	assert.Equal(t, 0, f.app.ledgerMismatches())
}

func (f checkoutFixture) cartItemID(session, variantID string) string {
	f.app.t.Helper()
	for _, s := range cartStores(f.app.do("GET", "/api/v1/cart", "", session)) {
		for _, it := range s.(map[string]any)["items"].([]any) {
			if m := it.(map[string]any); m["variantId"] == variantID {
				return m["id"].(string)
			}
		}
	}
	f.app.t.Fatalf("variant %s not in cart", variantID)
	return ""
}
