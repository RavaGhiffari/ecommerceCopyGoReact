//go:build integration

// End-to-end tests of Phase 5c (payment, cancellation, expiry, refund)
// against marketplace_test.
package router_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// placed is a checkout of 2 × Earbuds (store A, stock 5) and 3 × Mouse
// (store B, stock 200).
type placed struct {
	id, paymentID string
	orderA        string // the store A order id
	orderB        string
}

func (f checkoutFixture) placeTwoStores(t *testing.T) placed {
	t.Helper()
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 2).status)
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantB, 3).status)
	res := f.placeAll(f.buyer, f.address, f.storeA, f.storeB)
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	d := res.data()
	ords := d["orders"].([]any)
	return placed{
		id:        d["id"].(string),
		paymentID: d["payment"].(map[string]any)["id"].(string),
		orderA:    ords[0].(map[string]any)["id"].(string), // ordered by store name: Alpha, Beta
		orderB:    ords[1].(map[string]any)["id"].(string),
	}
}

func (f checkoutFixture) simulate(session, paymentID, outcome, eventID string) result {
	f.app.t.Helper()
	return f.app.do("POST", "/api/v1/payments/"+paymentID+"/simulate",
		fmt.Sprintf(`{"outcome":%q,"eventId":%q}`, outcome, eventID), session)
}

// stock returns (available, reserved) of a variant.
func (a *testApp) stock(variantID string) (int, int) {
	return a.count("SELECT available_quantity FROM inventory WHERE variant_id = $1", variantID),
		a.count("SELECT reserved_quantity FROM inventory WHERE variant_id = $1", variantID)
}

func (a *testApp) soldCount(productID string) int {
	return a.count("SELECT sold_count FROM products WHERE id = $1", productID)
}

func (a *testApp) movements(kind string) int {
	return a.count("SELECT count(*) FROM inventory_movements WHERE movement_type = $1", kind)
}

func orderStatuses(r result) []string {
	var out []string
	for _, o := range r.data()["orders"].([]any) {
		out = append(out, o.(map[string]any)["status"].(string))
	}
	return out
}

func errorDetails(r result) map[string]any {
	d, _ := r.body["error"].(map[string]any)["details"].(map[string]any)
	return d
}

func TestPayment_SuccessConsumesStockAndCountsSales(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.placeTwoStores(t)
	event := uuid.NewString()

	res := f.simulate(f.buyer, p.paymentID, "success", event)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	d := res.data()
	assert.Equal(t, "paid", d["status"])
	assert.NotNil(t, d["paidAt"])
	assert.Equal(t, "succeeded", d["payment"].(map[string]any)["status"])
	assert.NotNil(t, d["payment"].(map[string]any)["paidAt"])
	assert.Equal(t, []string{"paid", "paid"}, orderStatuses(res))

	// Reserved stock is consumed; available does not change again.
	avail, reserved := f.app.stock(f.variantA)
	assert.Equal(t, [2]int{3, 0}, [2]int{avail, reserved})
	avail, reserved = f.app.stock(f.variantB)
	assert.Equal(t, [2]int{197, 0}, [2]int{avail, reserved})
	assert.Equal(t, 2, f.app.soldCount(f.productA))
	assert.Equal(t, 3, f.app.soldCount(f.productB))
	assert.Equal(t, 2, f.app.movements("consumption"))
	assert.Equal(t, 0, f.app.ledgerMismatches())

	// History: pending_payment -> paid by the system (actor NULL).
	assert.Equal(t, 2, f.app.count(
		"SELECT count(*) FROM order_status_histories WHERE from_status = 'pending_payment' AND to_status = 'paid' AND actor_id IS NULL"))
	one := f.app.do("GET", "/api/v1/orders/"+p.orderA, "", f.buyer)
	assert.Len(t, one.data()["history"], 2)

	// The gateway delivers the same notification again: no second effect.
	again := f.simulate(f.buyer, p.paymentID, "success", event)
	require.Equal(t, http.StatusOK, again.status, again.raw)
	assert.Equal(t, "This payment event was already processed", again.body["message"])
	assert.Equal(t, 2, f.app.soldCount(f.productA))
	assert.Equal(t, 2, f.app.movements("consumption"))
	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM payment_events"))

	// The same event id with another outcome, and a new event after payment.
	conflict := f.simulate(f.buyer, p.paymentID, "failure", event)
	assert.Equal(t, http.StatusConflict, conflict.status)
	assert.Equal(t, "PAYMENT_EVENT_CONFLICT", conflict.errorCode())
	late := f.simulate(f.buyer, p.paymentID, "failure", uuid.NewString())
	assert.Equal(t, http.StatusConflict, late.status)
	assert.Equal(t, "CHECKOUT_NOT_PENDING", late.errorCode())
	assert.Equal(t, "paid", f.app.do("GET", "/api/v1/checkout/"+p.id, "", f.buyer).data()["status"])
}

func TestPayment_FailureCancelsAndReleasesStock(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.placeTwoStores(t)

	res := f.simulate(f.buyer, p.paymentID, "failure", uuid.NewString())
	require.Equal(t, http.StatusOK, res.status, res.raw)
	d := res.data()
	assert.Equal(t, "cancelled", d["status"])
	assert.NotNil(t, d["cancelledAt"])
	assert.Equal(t, "failed", d["payment"].(map[string]any)["status"])
	assert.Equal(t, []string{"cancelled", "cancelled"}, orderStatuses(res))

	avail, reserved := f.app.stock(f.variantA)
	assert.Equal(t, [2]int{5, 0}, [2]int{avail, reserved}, "stock is back on sale")
	assert.Equal(t, 0, f.app.soldCount(f.productA))
	assert.Equal(t, 2, f.app.movements("release"))
	assert.Equal(t, 0, f.app.ledgerMismatches())
}

func TestPayment_RequestRulesAndOwnership(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.placeTwoStores(t)

	bad := f.app.do("POST", "/api/v1/payments/"+p.paymentID+"/simulate", `{"outcome":"maybe"}`, f.buyer)
	assert.Equal(t, http.StatusUnprocessableEntity, bad.status)
	assert.Contains(t, errorDetails(bad), "outcome")
	assert.Contains(t, errorDetails(bad), "eventId")

	other := f.app.register("Orang Lain", "lain@example.com", "rahasia-123")
	res := f.simulate(other, p.paymentID, "success", uuid.NewString())
	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, "PAYMENT_NOT_FOUND", res.errorCode())
	assert.Equal(t, http.StatusNotFound, f.simulate(f.buyer, "not-a-uuid", "success", uuid.NewString()).status)
	assert.Equal(t, http.StatusUnauthorized, f.simulate("", p.paymentID, "success", uuid.NewString()).status)

	// Nothing happened to the checkout.
	assert.Equal(t, "pending_payment", f.app.do("GET", "/api/v1/checkout/"+p.id, "", f.buyer).data()["status"])
}

func TestPayment_AfterExpiryIsRejectedAndReleasesStock(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.placeTwoStores(t)
	// Time runs out, but the sweeper has not come by yet.
	_, err := f.app.pool.Exec(context.Background(), "UPDATE checkouts SET expires_at = NOW() - interval '1 second' WHERE id = $1", p.id)
	require.NoError(t, err)

	res := f.simulate(f.buyer, p.paymentID, "success", uuid.NewString())
	assert.Equal(t, http.StatusConflict, res.status, res.raw)
	assert.Equal(t, "CHECKOUT_EXPIRED", res.errorCode())

	// The expiry itself was committed: closed, stock released, no event kept.
	got := f.app.do("GET", "/api/v1/checkout/"+p.id, "", f.buyer)
	assert.Equal(t, "expired", got.data()["status"])
	assert.Equal(t, "expired", got.data()["payment"].(map[string]any)["status"])
	assert.Equal(t, []string{"cancelled", "cancelled"}, orderStatuses(got))
	avail, reserved := f.app.stock(f.variantA)
	assert.Equal(t, [2]int{5, 0}, [2]int{avail, reserved})
	assert.Equal(t, 0, f.app.soldCount(f.productA))
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM payment_events"))
	assert.Equal(t, 0, f.app.ledgerMismatches())
}

func TestCheckout_SweeperExpiresOnlyDueCheckouts(t *testing.T) {
	f := newCheckoutFixture(t)
	due := f.placeTwoStores(t)
	fresh := f.placeTwoStores(t)
	_, err := f.app.pool.Exec(context.Background(), "UPDATE checkouts SET expires_at = NOW() - interval '1 minute' WHERE id = $1", due.id)
	require.NoError(t, err)

	n, err := f.app.checkout.ExpireDue(context.Background(), 50)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	assert.Equal(t, "expired", f.app.do("GET", "/api/v1/checkout/"+due.id, "", f.buyer).data()["status"])
	assert.Equal(t, "pending_payment", f.app.do("GET", "/api/v1/checkout/"+fresh.id, "", f.buyer).data()["status"])
	avail, reserved := f.app.stock(f.variantA)
	assert.Equal(t, [2]int{3, 2}, [2]int{avail, reserved}, "only the fresh checkout still holds stock")
	assert.Equal(t, 1, f.app.count(
		"SELECT count(*) FROM order_status_histories WHERE order_id = $1 AND to_status = 'cancelled' AND note = 'Payment time ran out' AND actor_id IS NULL", due.orderA))

	// Nothing left to do: a second sweep is a no-op.
	n, err = f.app.checkout.ExpireDue(context.Background(), 50)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, f.app.ledgerMismatches())
}

func TestCheckout_BuyerCancelsWhileUnpaid(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.placeTwoStores(t)

	// One unpaid order cannot be cancelled on its own: the payment covers both.
	single := f.app.do("POST", "/api/v1/orders/"+p.orderA+"/cancel", "", f.buyer)
	assert.Equal(t, http.StatusConflict, single.status, single.raw)
	assert.Equal(t, "INVALID_STATUS_TRANSITION", single.errorCode())
	assert.Equal(t, p.id, errorDetails(single)["checkoutId"])

	other := f.app.register("Orang Lain", "lain@example.com", "rahasia-123")
	assert.Equal(t, http.StatusNotFound, f.app.do("POST", "/api/v1/checkout/"+p.id+"/cancel", "", other).status)

	res := f.app.do("POST", "/api/v1/checkout/"+p.id+"/cancel", "", f.buyer)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, "cancelled", res.data()["status"])
	assert.Equal(t, "cancelled", res.data()["payment"].(map[string]any)["status"])
	assert.Equal(t, []string{"cancelled", "cancelled"}, orderStatuses(res))
	avail, reserved := f.app.stock(f.variantA)
	assert.Equal(t, [2]int{5, 0}, [2]int{avail, reserved})
	assert.Equal(t, 2, f.app.count(
		"SELECT count(*) FROM order_status_histories WHERE to_status = 'cancelled' AND note = 'Cancelled by the buyer' AND actor_id IS NOT NULL"))

	again := f.app.do("POST", "/api/v1/checkout/"+p.id+"/cancel", "", f.buyer)
	assert.Equal(t, http.StatusConflict, again.status)
	assert.Equal(t, "CHECKOUT_NOT_PENDING", again.errorCode())
	pay := f.simulate(f.buyer, p.paymentID, "success", uuid.NewString())
	assert.Equal(t, "CHECKOUT_NOT_PENDING", pay.errorCode(), "a cancelled checkout cannot be paid")
	assert.Equal(t, 0, f.app.ledgerMismatches())
}

func TestOrders_RefundAfterPaymentRestocks(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.placeTwoStores(t)
	require.Equal(t, http.StatusOK, f.simulate(f.buyer, p.paymentID, "success", uuid.NewString()).status)

	other := f.app.register("Orang Lain", "lain@example.com", "rahasia-123")
	assert.Equal(t, http.StatusNotFound, f.app.do("POST", "/api/v1/orders/"+p.orderA+"/cancel", "", other).status)

	res := f.app.do("POST", "/api/v1/orders/"+p.orderA+"/cancel", "", f.buyer)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, "refunded", res.data()["status"])
	hist := res.data()["history"].([]any)
	assert.Equal(t, "refunded", hist[len(hist)-1].(map[string]any)["toStatus"])

	// Store A's units are back on sale and no longer counted as sold;
	// store B's order is untouched.
	avail, reserved := f.app.stock(f.variantA)
	assert.Equal(t, [2]int{5, 0}, [2]int{avail, reserved})
	assert.Equal(t, 0, f.app.soldCount(f.productA))
	assert.Equal(t, 3, f.app.soldCount(f.productB))
	assert.Equal(t, 1, f.app.count(
		"SELECT count(*) FROM inventory_movements WHERE movement_type = 'restock' AND reference_type = 'order' AND reference_id = $1", p.orderA))
	assert.Equal(t, "paid", f.app.do("GET", "/api/v1/orders/"+p.orderB, "", f.buyer).data()["status"])
	assert.Equal(t, 0, f.app.ledgerMismatches())

	// Final status: a second cancel is refused.
	again := f.app.do("POST", "/api/v1/orders/"+p.orderA+"/cancel", "", f.buyer)
	assert.Equal(t, http.StatusConflict, again.status)
	assert.Equal(t, "INVALID_STATUS_TRANSITION", again.errorCode())

	// Once the seller is processing it (5d adds the endpoint), it is too late.
	_, err := f.app.pool.Exec(context.Background(), "UPDATE orders SET status = 'processing' WHERE id = $1", p.orderB)
	require.NoError(t, err)
	late := f.app.do("POST", "/api/v1/orders/"+p.orderB+"/cancel", "", f.buyer)
	assert.Equal(t, http.StatusConflict, late.status)
	assert.Equal(t, map[string]any{"from": "processing", "to": "refunded"}, errorDetails(late))
	assert.Equal(t, 3, f.app.soldCount(f.productB))
}

// Several lifecycle requests for one checkout at the same moment: payments
// with different event ids, a failure and a cancel. The checkout row lock
// makes them run one at a time, so exactly one wins and the rest see that it
// is no longer pending. Repeated a few times to give the race a chance.
func TestCheckout_ConcurrentLifecycleRequestsHaveOneWinner(t *testing.T) {
	f := newCheckoutFixture(t)
	for round := range 5 {
		// Enough stock for every round (a paid round keeps its 2 units).
		require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":50}`, f.sellerA).status)
		p := f.placeTwoStores(t)

		type call func() result
		calls := []call{
			func() result { return f.simulate(f.buyer, p.paymentID, "success", uuid.NewString()) },
			func() result { return f.simulate(f.buyer, p.paymentID, "success", uuid.NewString()) },
			func() result { return f.simulate(f.buyer, p.paymentID, "failure", uuid.NewString()) },
			func() result { return f.app.do("POST", "/api/v1/checkout/"+p.id+"/cancel", "", f.buyer) },
		}
		statuses := map[int]int{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		start := make(chan struct{})
		for _, c := range calls {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				res := c()
				mu.Lock()
				statuses[res.status]++
				mu.Unlock()
				if res.status == http.StatusConflict {
					assert.Equal(t, "CHECKOUT_NOT_PENDING", res.errorCode())
				}
			}()
		}
		close(start)
		wg.Wait()

		require.Equal(t, map[int]int{http.StatusOK: 1, http.StatusConflict: 3}, statuses, "round %d", round)
		// Exactly one outcome was recorded, and it is consistent.
		var status, payStatus string
		require.NoError(t, f.app.pool.QueryRow(context.Background(),
			"SELECT c.status, p.status FROM checkouts c JOIN payments p ON p.checkout_id = c.id WHERE c.id = $1", p.id).Scan(&status, &payStatus))
		events := f.app.count("SELECT count(*) FROM payment_events WHERE payment_id = $1", p.paymentID)
		switch status + "/" + payStatus {
		case "paid/succeeded", "cancelled/failed":
			assert.Equal(t, 1, events, "round %d", round)
		case "cancelled/cancelled":
			assert.Equal(t, 0, events, "round %d", round)
		default:
			t.Fatalf("round %d: unexpected final state %s/%s", round, status, payStatus)
		}
		_, reserved := f.app.stock(f.variantA)
		assert.Equal(t, 0, reserved, "round %d: nothing stays reserved", round)
	}
	assert.Equal(t, 0, f.app.ledgerMismatches())
}

// The sweeper and a payment reach an expired checkout at the same moment.
// Whoever locks the row first closes it as expired; the payment is refused
// either way, and the stock is released exactly once.
func TestCheckout_PaymentRacingTheSweeper(t *testing.T) {
	f := newCheckoutFixture(t)
	for round := range 5 {
		// Enough stock for every round (a paid round keeps its 2 units).
		require.Equal(t, http.StatusOK, f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":50}`, f.sellerA).status)
		p := f.placeTwoStores(t)
		_, err := f.app.pool.Exec(context.Background(), "UPDATE checkouts SET expires_at = NOW() - interval '1 second' WHERE id = $1", p.id)
		require.NoError(t, err)

		var pay result
		var sweepErr error
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); pay = f.simulate(f.buyer, p.paymentID, "success", uuid.NewString()) }()
		go func() { defer wg.Done(); _, sweepErr = f.app.checkout.ExpireDue(context.Background(), 50) }()
		wg.Wait()

		require.NoError(t, sweepErr)
		assert.Equal(t, http.StatusConflict, pay.status, "round %d: %s", round, pay.raw)
		assert.Contains(t, []string{"CHECKOUT_EXPIRED", "CHECKOUT_NOT_PENDING"}, pay.errorCode())
		assert.Equal(t, "expired", f.app.do("GET", "/api/v1/checkout/"+p.id, "", f.buyer).data()["status"])
		assert.Equal(t, 2, f.app.count(
			"SELECT count(*) FROM inventory_movements WHERE movement_type = 'release' AND reference_id = $1", p.id),
			"round %d: one release per line, not two", round)
	}
	assert.Equal(t, 0, f.app.soldCount(f.productA))
	assert.Equal(t, 0, f.app.ledgerMismatches())
}
