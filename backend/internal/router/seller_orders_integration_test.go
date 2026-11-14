//go:build integration

// End-to-end tests of Phase 5d (seller orders) against marketplace_test.
package router_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f checkoutFixture) setStatus(session, orderID, body string) result {
	f.app.t.Helper()
	return f.app.do("PATCH", "/api/v1/seller/orders/"+orderID+"/status", body, session)
}

func (f checkoutFixture) paidCheckout(t *testing.T) placed {
	t.Helper()
	p := f.placeTwoStores(t)
	require.Equal(t, http.StatusOK, f.simulate(f.buyer, p.paymentID, "success", uuid.NewString()).status)
	return p
}

func TestSellerOrders_EachStoreSeesOnlyItsOwn(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.paidCheckout(t)        // one order per store
	unpaid := f.placeTwoStores(t) // and a second, unpaid checkout

	listA := f.app.do("GET", "/api/v1/seller/orders", "", f.sellerA)
	require.Equal(t, http.StatusOK, listA.status, listA.raw)
	assert.Equal(t, float64(2), listA.data()["total"])
	for _, it := range listItems(listA) {
		assert.Equal(t, "Toko Alpha", it.(map[string]any)["store"].(map[string]any)["name"])
	}
	assert.Equal(t, unpaid.orderA, listItems(listA)[0].(map[string]any)["id"], "newest first")

	// Filter and pagination.
	paid := f.app.do("GET", "/api/v1/seller/orders?status=paid", "", f.sellerA)
	require.Equal(t, http.StatusOK, paid.status)
	require.Len(t, listItems(paid), 1)
	assert.Equal(t, p.orderA, listItems(paid)[0].(map[string]any)["id"])
	page2 := f.app.do("GET", "/api/v1/seller/orders?pageSize=1&page=2", "", f.sellerA)
	assert.Equal(t, p.orderA, listItems(page2)[0].(map[string]any)["id"])
	assert.Equal(t, float64(2), page2.data()["totalPages"])
	bad := f.app.do("GET", "/api/v1/seller/orders?status=lost", "", f.sellerA)
	assert.Equal(t, http.StatusUnprocessableEntity, bad.status)

	// Detail: items, address snapshot and history.
	one := f.app.do("GET", "/api/v1/seller/orders/"+p.orderA, "", f.sellerA)
	require.Equal(t, http.StatusOK, one.status, one.raw)
	assert.Equal(t, "Earbuds", one.data()["items"].([]any)[0].(map[string]any)["productName"])
	assert.Equal(t, "Bandung", one.data()["shippingAddress"].(map[string]any)["city"])
	assert.Len(t, one.data()["history"], 2)

	// Store B's orders and other people's requests.
	for _, path := range []string{"/api/v1/seller/orders/" + p.orderB, "/api/v1/seller/orders/not-a-uuid"} {
		res := f.app.do("GET", path, "", f.sellerA)
		assert.Equal(t, http.StatusNotFound, res.status, path)
		assert.Equal(t, "ORDER_NOT_FOUND", res.errorCode())
	}
	assert.Equal(t, http.StatusForbidden, f.app.do("GET", "/api/v1/seller/orders", "", f.buyer).status, "a buyer is not a seller")
	assert.Equal(t, http.StatusUnauthorized, f.app.do("GET", "/api/v1/seller/orders", "", "").status)
}

func TestSellerOrders_FulfilmentFlow(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.paidCheckout(t)

	for _, step := range []struct{ status, note string }{
		{"processing", ""},
		{"shipped", "Resi JNE 0123456789"},
		{"delivered", ""},
	} {
		body := fmt.Sprintf(`{"status":%q}`, step.status)
		if step.note != "" {
			body = fmt.Sprintf(`{"status":%q,"note":%q}`, step.status, step.note)
		}
		res := f.setStatus(f.sellerA, p.orderA, body)
		require.Equal(t, http.StatusOK, res.status, "%s: %s", step.status, res.raw)
		assert.Equal(t, step.status, res.data()["status"])
	}

	// The buyer sees the same order and its whole history.
	got := f.app.do("GET", "/api/v1/orders/"+p.orderA, "", f.buyer)
	require.Equal(t, http.StatusOK, got.status)
	assert.Equal(t, "delivered", got.data()["status"])
	var trail []string
	for _, h := range got.data()["history"].([]any) {
		trail = append(trail, h.(map[string]any)["toStatus"].(string))
	}
	assert.Equal(t, []string{"pending_payment", "paid", "processing", "shipped", "delivered"}, trail)
	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM order_status_histories WHERE order_id = $1 AND note = 'Resi JNE 0123456789'", p.orderA))
	assert.Equal(t, 3, f.app.count(`
		SELECT count(*) FROM order_status_histories h JOIN stores s ON s.owner_id = h.actor_id
		WHERE h.order_id = $1 AND s.name = 'Toko Alpha'`, p.orderA), "the seller is the actor of their changes")

	// Too late for the buyer to cancel; the other store's order is untouched.
	assert.Equal(t, http.StatusConflict, f.app.do("POST", "/api/v1/orders/"+p.orderA+"/cancel", "", f.buyer).status)
	assert.Equal(t, "paid", f.app.do("GET", "/api/v1/orders/"+p.orderB, "", f.buyer).data()["status"])
}

func TestSellerOrders_TransitionRules(t *testing.T) {
	f := newCheckoutFixture(t)
	unpaid := f.placeTwoStores(t)
	p := f.paidCheckout(t)

	cases := []struct {
		name, order, body string
		status            int
		code              string
	}{
		{"unpaid order", unpaid.orderA, `{"status":"processing"}`, http.StatusConflict, "INVALID_STATUS_TRANSITION"},
		{"skipping processing", p.orderA, `{"status":"shipped"}`, http.StatusConflict, "INVALID_STATUS_TRANSITION"},
		{"refund is the buyer's", p.orderA, `{"status":"refunded"}`, http.StatusConflict, "INVALID_STATUS_TRANSITION"},
		{"cancel is not a seller action", p.orderA, `{"status":"cancelled"}`, http.StatusConflict, "INVALID_STATUS_TRANSITION"},
		{"unknown status", p.orderA, `{"status":"teleported"}`, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"missing status", p.orderA, `{}`, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"unknown field", p.orderA, `{"status":"processing","storeId":"x"}`, http.StatusBadRequest, ""},
		{"another store's order", p.orderB, `{"status":"processing"}`, http.StatusNotFound, "ORDER_NOT_FOUND"},
	}
	for _, c := range cases {
		res := f.setStatus(f.sellerA, c.order, c.body)
		assert.Equal(t, c.status, res.status, "%s: %s", c.name, res.raw)
		if c.code != "" {
			assert.Equal(t, c.code, res.errorCode(), c.name)
		}
	}
	skip := f.setStatus(f.sellerA, p.orderA, `{"status":"shipped"}`)
	assert.Equal(t, map[string]any{"from": "paid", "to": "shipped"}, errorDetails(skip))

	// None of the refused requests changed anything.
	assert.Equal(t, "paid", f.app.do("GET", "/api/v1/seller/orders/"+p.orderA, "", f.sellerA).data()["status"])
	assert.Equal(t, "paid", f.app.do("GET", "/api/v1/seller/orders/"+p.orderB, "", f.sellerB).data()["status"])
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM order_status_histories WHERE to_status IN ('processing', 'shipped')"))

	// Final statuses stay final.
	for _, s := range []string{"processing", "shipped", "delivered"} {
		require.Equal(t, http.StatusOK, f.setStatus(f.sellerA, p.orderA, `{"status":"`+s+`"}`).status)
	}
	again := f.setStatus(f.sellerA, p.orderA, `{"status":"delivered"}`)
	assert.Equal(t, http.StatusConflict, again.status)
}

// The buyer cancels (refund) while the seller starts processing the same
// paid order. Both lock the order row, so exactly one of them wins and the
// stock is restocked only if the refund won. Repeated a few times.
func TestSellerOrders_ProcessingRacesTheBuyersCancel(t *testing.T) {
	f := newCheckoutFixture(t)
	refunds := 0
	for round := range 5 {
		require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":50}`, f.sellerA).status)
		p := f.paidCheckout(t)

		var cancel, process result
		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			cancel = f.app.do("POST", "/api/v1/orders/"+p.orderA+"/cancel", "", f.buyer)
		}()
		go func() {
			defer wg.Done()
			<-start
			process = f.setStatus(f.sellerA, p.orderA, `{"status":"processing"}`)
		}()
		close(start)
		wg.Wait()

		statuses := []int{cancel.status, process.status}
		assert.ElementsMatch(t, []int{http.StatusOK, http.StatusConflict}, statuses, "round %d: %s | %s", round, cancel.raw, process.raw)
		final := f.app.do("GET", "/api/v1/orders/"+p.orderA, "", f.buyer).data()["status"]
		if cancel.status == http.StatusOK {
			refunds++
			assert.Equal(t, "refunded", final)
			assert.Equal(t, 1, f.app.count("SELECT count(*) FROM inventory_movements WHERE movement_type = 'restock' AND reference_id = $1", p.orderA))
		} else {
			assert.Equal(t, "processing", final)
			assert.Equal(t, 0, f.app.count("SELECT count(*) FROM inventory_movements WHERE movement_type = 'restock' AND reference_id = $1", p.orderA))
		}
	}
	t.Logf("refund won %d of 5 rounds", refunds)
	assert.Equal(t, 0, f.app.ledgerMismatches())
}
