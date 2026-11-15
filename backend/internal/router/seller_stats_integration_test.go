//go:build integration

// End-to-end tests of Phase 6c (seller dashboard) against marketplace_test.
package router_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f checkoutFixture) stats(session string) result {
	f.app.t.Helper()
	return f.app.do("GET", "/api/v1/seller/stats", "", session)
}

func (a *testApp) exec(sql string, args ...any) {
	a.t.Helper()
	_, err := a.pool.Exec(context.Background(), sql, args...)
	require.NoError(a.t, err)
}

func TestSellerStats_EmptyStore(t *testing.T) {
	f := newCheckoutFixture(t)

	res := f.stats(f.sellerA)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	d := res.data()
	byStatus := d["ordersByStatus"].(map[string]any)
	assert.Len(t, byStatus, 7, "every status is present")
	for s, n := range byStatus {
		assert.Equal(t, float64(0), n, s)
	}
	assert.Equal(t, map[string]any{"last30Days": "0.00", "allTime": "0.00"}, d["revenue"])
	assert.Equal(t, map[string]any{"last30Days": float64(0), "allTime": float64(0)}, d["unitsSold"])
	assert.Equal(t, []any{}, d["topProducts"], "an empty list, not null")
	assert.Equal(t, map[string]any{"average": float64(0), "count": float64(0)}, d["rating"])
	assert.NotEmpty(t, d["since"])
}

// The fixture below, store A (Earbuds 100000.50, Kabel 10000):
//
//	p1 paid → delivered            Earbuds ×2            200001.00  counted
//	p2 placed, never paid          Earbuds ×2                       not counted (pending)
//	p3 paid → refunded by buyer    Earbuds ×2                       not counted (refunded)
//	p4 paid                        Earbuds ×2 + Kabel ×5 250001.00  counted
//	p5 paid, created 40 days ago   Earbuds ×2            200001.00  all time only
//
// Every checkout also buys Mouse ×3 (50000) from store B.
func TestSellerStats_MatchesHandComputedNumbers(t *testing.T) {
	f := newCheckoutFixture(t)
	restock := f.app.do("PUT", f.variantAPath+"/stock", `{"quantity":50}`, f.sellerA)
	require.Equal(t, http.StatusOK, restock.status, restock.raw)
	var categoryID string
	require.NoError(t, f.app.pool.QueryRow(context.Background(), "SELECT category_id FROM products WHERE id = $1", f.productA).Scan(&categoryID))
	kabel := f.app.createProduct(f.sellerA, productBody("Kabel", categoryID, variantBody{"Putih", "AL-2", "10000", 50}))
	kabelVariant := kabel["variants"].([]any)[0].(map[string]any)["id"].(string)

	p1 := f.paidCheckout(t)
	for _, s := range []string{"processing", "shipped", "delivered"} {
		require.Equal(t, http.StatusOK, f.setStatus(f.sellerA, p1.orderA, `{"status":"`+s+`"}`).status)
	}
	f.placeTwoStores(t) // p2: pending
	p3 := f.paidCheckout(t)
	require.Equal(t, http.StatusOK, f.app.do("POST", "/api/v1/orders/"+p3.orderA+"/cancel", "", f.buyer).status)
	require.Equal(t, http.StatusOK, f.add(f.buyer, kabelVariant, 5).status)
	f.paidCheckout(t) // p4
	p5 := f.paidCheckout(t)
	f.app.exec("UPDATE orders SET created_at = now() - interval '40 days' WHERE checkout_id = $1", p5.id)
	// Ratings come from the review counters (6a); set them directly here.
	f.app.exec("UPDATE products SET rating_count = 2, rating_total = 9 WHERE id = $1", f.productA)

	res := f.stats(f.sellerA)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	d := res.data()
	assert.Equal(t, map[string]any{
		"pending_payment": float64(1), "paid": float64(2), "processing": float64(0), "shipped": float64(0),
		"delivered": float64(1), "cancelled": float64(0), "refunded": float64(1),
	}, d["ordersByStatus"])
	assert.Equal(t, map[string]any{"last30Days": "450002.00", "allTime": "650003.00"}, d["revenue"], "items only, no shipping")
	assert.Equal(t, map[string]any{"last30Days": float64(9), "allTime": float64(11)}, d["unitsSold"])
	assert.Equal(t, []any{
		map[string]any{"productId": kabel["id"], "name": "Kabel", "status": "active", "units": float64(5), "revenue": "50000.00"},
		map[string]any{"productId": f.productA, "name": "Earbuds", "status": "active", "units": float64(4), "revenue": "400002.00"},
	}, d["topProducts"], "by units in the window; the 40-day-old and refunded orders are left out")
	assert.Equal(t, map[string]any{"average": 4.5, "count": float64(2)}, d["rating"])

	// Store B only sees its own orders: Mouse ×3 in p1, p3, p4, p5 (all paid,
	// p5 old) and p2 pending. Store A's refund does not touch B's order.
	b := f.stats(f.sellerB).data()
	assert.Equal(t, map[string]any{"last30Days": "450000.00", "allTime": "600000.00"}, b["revenue"])
	assert.Equal(t, float64(4), b["ordersByStatus"].(map[string]any)["paid"])
	assert.Equal(t, float64(1), b["ordersByStatus"].(map[string]any)["pending_payment"])
	assert.Equal(t, map[string]any{"average": float64(0), "count": float64(0)}, b["rating"], "store A's rating is not B's")
	assert.Len(t, b["topProducts"], 1)
}

func TestSellerStats_Access(t *testing.T) {
	f := newCheckoutFixture(t)
	assert.Equal(t, http.StatusUnauthorized, f.stats("").status)
	assert.Equal(t, http.StatusForbidden, f.stats(f.buyer).status, "a buyer is not a seller")
	// The route takes no store id: there is nothing to swap for another store's.
	assert.Equal(t, http.StatusOK, f.app.do("GET", "/api/v1/seller/stats?storeId="+uuid.NewString(), "", f.sellerA).status)
}
