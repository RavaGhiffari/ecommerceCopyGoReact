//go:build integration

// End-to-end tests of Phase 6a (reviews) against marketplace_test.
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

// deliveredItem buys one unit of a variant in its own checkout, pays, lets the
// seller deliver it, and returns the order id and the order item id.
func (f checkoutFixture) deliveredItem(t *testing.T, variantID, storeID, seller string) (orderID, itemID string) {
	t.Helper()
	require.Equal(t, http.StatusOK, f.add(f.buyer, variantID, 1).status)
	res := f.placeAll(f.buyer, f.address, storeID)
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	paymentID := res.data()["payment"].(map[string]any)["id"].(string)
	orderID = res.data()["orders"].([]any)[0].(map[string]any)["id"].(string)
	require.Equal(t, http.StatusOK, f.simulate(f.buyer, paymentID, "success", uuid.NewString()).status)
	for _, s := range []string{"processing", "shipped", "delivered"} {
		r := f.setStatus(seller, orderID, fmt.Sprintf(`{"status":%q}`, s))
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	order := f.app.do("GET", "/api/v1/orders/"+orderID, "", f.buyer)
	itemID = order.data()["items"].([]any)[0].(map[string]any)["id"].(string)
	return orderID, itemID
}

func (f checkoutFixture) review(session, itemID string, rating int, comment string) result {
	f.app.t.Helper()
	return f.app.do("POST", "/api/v1/reviews",
		fmt.Sprintf(`{"orderItemId":%q,"rating":%d,"comment":%q}`, itemID, rating, comment), session)
}

// ratingMismatches counts products whose counters differ from their visible
// reviews. The invariant: always 0.
func (a *testApp) ratingMismatches() int {
	return a.count(`
		SELECT count(*) FROM products p
		LEFT JOIN (SELECT product_id, COUNT(*) n, SUM(rating) s FROM reviews WHERE status = 'visible' GROUP BY product_id) r
		       ON r.product_id = p.id
		WHERE p.rating_count <> COALESCE(r.n, 0) OR p.rating_total <> COALESCE(r.s, 0)`)
}

func productRating(t *testing.T, r result) (float64, float64) {
	t.Helper()
	rt := r.data()["rating"].(map[string]any)
	return rt["average"].(float64), rt["count"].(float64)
}

func TestReviews_OnlyTheBuyersDeliveredItems(t *testing.T) {
	f := newCheckoutFixture(t)
	p := f.paidCheckout(t) // order A and order B, both paid

	itemA := f.app.do("GET", "/api/v1/orders/"+p.orderA, "", f.buyer).data()["items"].([]any)[0].(map[string]any)["id"].(string)
	itemB := f.app.do("GET", "/api/v1/orders/"+p.orderB, "", f.buyer).data()["items"].([]any)[0].(map[string]any)["id"].(string)

	// Paid is not delivered.
	res := f.review(f.buyer, itemA, 5, "")
	require.Equal(t, http.StatusConflict, res.status, res.raw)
	assert.Equal(t, "REVIEW_NOT_ALLOWED", res.errorCode())
	assert.Equal(t, "paid", res.body["error"].(map[string]any)["details"].(map[string]any)["orderStatus"])

	// A refunded order never becomes reviewable.
	require.Equal(t, http.StatusOK, f.app.do("POST", "/api/v1/orders/"+p.orderB+"/cancel", "", f.buyer).status)
	res = f.review(f.buyer, itemB, 5, "")
	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "refunded", res.body["error"].(map[string]any)["details"].(map[string]any)["orderStatus"])

	for _, s := range []string{"processing", "shipped", "delivered"} {
		require.Equal(t, http.StatusOK, f.setStatus(f.sellerA, p.orderA, fmt.Sprintf(`{"status":%q}`, s)).status)
	}

	// Waiting for review: only the delivered item.
	pending := f.app.do("GET", "/api/v1/me/reviews/pending", "", f.buyer)
	require.Equal(t, http.StatusOK, pending.status, pending.raw)
	require.Len(t, listItems(pending), 1)
	first := listItems(pending)[0].(map[string]any)
	assert.Equal(t, itemA, first["orderItemId"])
	assert.Equal(t, "Earbuds", first["productName"])
	assert.Equal(t, "Toko Alpha", first["storeName"])

	// Nobody else can review it: another buyer, the seller, anonymous.
	other := f.app.register("Orang Lain", "lain@example.com", "rahasia-123")
	for _, who := range []string{other, f.sellerA} {
		res := f.review(who, itemA, 5, "")
		assert.Equal(t, http.StatusNotFound, res.status, res.raw)
		assert.Equal(t, "ORDER_ITEM_NOT_FOUND", res.errorCode())
	}
	assert.Equal(t, http.StatusUnauthorized, f.review("", itemA, 5, "").status)

	// Validation and strict JSON.
	res = f.app.do("POST", "/api/v1/reviews", `{"orderItemId":"nope","rating":0}`, f.buyer)
	require.Equal(t, http.StatusUnprocessableEntity, res.status)
	details := res.body["error"].(map[string]any)["details"].(map[string]any)
	assert.Contains(t, details, "orderItemId")
	assert.Contains(t, details, "rating")
	res = f.app.do("POST", "/api/v1/reviews", fmt.Sprintf(`{"orderItemId":%q,"rating":5,"productId":%q}`, itemA, f.productB), f.buyer)
	assert.Equal(t, http.StatusBadRequest, res.status, "unknown fields are rejected")

	// The review itself.
	res = f.review(f.buyer, itemA, 4, "  Suaranya jernih  ")
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	assert.Equal(t, "Suaranya jernih", res.data()["comment"], "comment trimmed")
	assert.Equal(t, f.productA, res.data()["productId"], "product taken from the order, not the request")
	reviewID := res.data()["id"].(string)

	// Once per purchase.
	res = f.review(f.buyer, itemA, 5, "")
	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "REVIEW_EXISTS", res.errorCode())

	// It shows up everywhere it should.
	assert.Empty(t, listItems(f.app.do("GET", "/api/v1/me/reviews/pending", "", f.buyer)))
	mine := f.app.do("GET", "/api/v1/me/reviews", "", f.buyer)
	require.Len(t, listItems(mine), 1)
	assert.Equal(t, reviewID, listItems(mine)[0].(map[string]any)["id"])
	assert.NotEmpty(t, listItems(mine)[0].(map[string]any)["item"].(map[string]any)["orderNumber"])

	orderItem := f.app.do("GET", "/api/v1/orders/"+p.orderA, "", f.buyer).data()["items"].([]any)[0].(map[string]any)
	assert.Equal(t, map[string]any{"id": reviewID, "rating": float64(4)}, orderItem["review"])

	detail := f.app.do("GET", "/api/v1/stores/toko-alpha/products/earbuds", "", "")
	require.Equal(t, http.StatusOK, detail.status, detail.raw)
	avg, n := productRating(t, detail)
	assert.Equal(t, 4.0, avg)
	assert.Equal(t, 1.0, n)
	store := f.app.do("GET", "/api/v1/stores/toko-alpha", "", "")
	avg, n = productRating(t, store)
	assert.Equal(t, 4.0, avg)
	assert.Equal(t, 1.0, n)
	assert.Equal(t, 0, f.app.ratingMismatches())
}

func TestReviews_PublicListSummaryAndEdits(t *testing.T) {
	f := newCheckoutFixture(t)
	_, item1 := f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)
	_, item2 := f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)
	_, item3 := f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)

	r1 := f.review(f.buyer, item1, 5, "Enak dipakai")
	require.Equal(t, http.StatusCreated, r1.status, r1.raw)
	require.Equal(t, http.StatusCreated, f.review(f.buyer, item2, 2, "").status)
	r3 := f.review(f.buyer, item3, 4, "")
	require.Equal(t, http.StatusCreated, r3.status)
	id1, id3 := r1.data()["id"].(string), r3.data()["id"].(string)

	path := "/api/v1/products/" + f.productB + "/reviews"
	list := f.app.do("GET", path, "", "")
	require.Equal(t, http.StatusOK, list.status, list.raw)
	sum := list.data()["summary"].(map[string]any)
	assert.Equal(t, 3.7, sum["average"], "(5+2+4)/3 = 3.67 -> 3.7")
	assert.Equal(t, 3.0, sum["count"])
	assert.Equal(t, map[string]any{"1": 0.0, "2": 1.0, "3": 0.0, "4": 1.0, "5": 1.0}, sum["distribution"])
	items := listItems(list)
	require.Len(t, items, 3)
	newest := items[0].(map[string]any)
	assert.Equal(t, id3, newest["id"], "newest first")
	assert.Equal(t, "P***", newest["reviewerName"], `"Pembeli" is masked`)
	assert.Equal(t, "Hitam", newest["variantName"])
	assert.Equal(t, false, newest["edited"])
	assert.NotContains(t, list.raw, "pembeli@example.com")
	assert.NotContains(t, newest, "userId")

	// Pagination.
	page2 := f.app.do("GET", path+"?pageSize=2&page=2", "", "")
	require.Equal(t, http.StatusOK, page2.status)
	assert.Len(t, listItems(page2), 1)
	assert.Equal(t, 2.0, page2.data()["totalPages"])
	assert.Equal(t, http.StatusUnprocessableEntity, f.app.do("GET", path+"?pageSize=500", "", "").status)

	// Edit: the aggregate moves by the difference.
	edit := f.app.do("PATCH", "/api/v1/reviews/"+id1, `{"rating":1}`, f.buyer)
	require.Equal(t, http.StatusOK, edit.status, edit.raw)
	assert.Equal(t, "Enak dipakai", edit.data()["comment"], "omitted fields stay")
	assert.Equal(t, 7, f.app.count("SELECT rating_total FROM products WHERE id = $1", f.productB))
	assert.Equal(t, 3, f.app.count("SELECT rating_count FROM products WHERE id = $1", f.productB))
	edited := f.app.do("GET", path, "", "")
	assert.Equal(t, 2.3, edited.data()["summary"].(map[string]any)["average"])
	for _, it := range listItems(edited) {
		if it.(map[string]any)["id"] == id1 {
			assert.Equal(t, true, it.(map[string]any)["edited"])
		}
	}
	require.Equal(t, http.StatusOK, f.app.do("PATCH", "/api/v1/reviews/"+id1, `{"comment":"Berubah pikiran"}`, f.buyer).status)
	assert.Equal(t, 7, f.app.count("SELECT rating_total FROM products WHERE id = $1", f.productB), "a comment edit keeps the counters")

	// Bad edits.
	other := f.app.register("Orang Lain", "lain@example.com", "rahasia-123")
	for _, c := range []struct {
		who, id, body string
		status        int
	}{
		{f.buyer, id1, `{}`, http.StatusUnprocessableEntity},
		{f.buyer, id1, `{"rating":9}`, http.StatusUnprocessableEntity},
		{f.buyer, id1, `{"status":"hidden"}`, http.StatusBadRequest},
		{other, id1, `{"rating":5}`, http.StatusNotFound},
		{f.buyer, "not-a-uuid", `{"rating":5}`, http.StatusNotFound},
		{"", id1, `{"rating":5}`, http.StatusUnauthorized},
	} {
		res := f.app.do("PATCH", "/api/v1/reviews/"+c.id, c.body, c.who)
		assert.Equal(t, c.status, res.status, "%s %s", c.body, res.raw)
	}
	assert.Equal(t, 7, f.app.count("SELECT rating_total FROM products WHERE id = $1", f.productB), "refused edits change nothing")

	// A hidden review (an admin action from 6d, done by hand here) leaves the
	// public list and the aggregate; editing it does not touch the counters.
	_, err := f.app.pool.Exec(t.Context(), `
		WITH h AS (UPDATE reviews SET status = 'hidden' WHERE id = $1 RETURNING product_id, rating)
		UPDATE products p SET rating_count = rating_count - 1, rating_total = rating_total - h.rating FROM h WHERE p.id = h.product_id`, id3)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, f.app.do("PATCH", "/api/v1/reviews/"+id3, `{"rating":1}`, f.buyer).status)
	assert.Len(t, listItems(f.app.do("GET", path, "", "")), 2)
	var hiddenStatus string
	for _, it := range listItems(f.app.do("GET", "/api/v1/me/reviews", "", f.buyer)) {
		if it.(map[string]any)["id"] == id3 {
			hiddenStatus = it.(map[string]any)["status"].(string)
		}
	}
	assert.Equal(t, "hidden", hiddenStatus, "the author still sees it")
	assert.Equal(t, 0, f.app.ratingMismatches())

	// Not public -> 404.
	require.Equal(t, http.StatusNoContent, f.app.do("DELETE", "/api/v1/seller/products/"+f.productB, "", f.sellerB).status)
	res := f.app.do("GET", path, "", "")
	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, "PRODUCT_NOT_FOUND", res.errorCode())
	assert.Equal(t, http.StatusNotFound, f.app.do("GET", "/api/v1/products/not-a-uuid/reviews", "", "").status)
}

func TestReviews_SortByRating(t *testing.T) {
	f := newCheckoutFixture(t)
	var komputer string
	require.NoError(t, f.app.pool.QueryRow(t.Context(), "SELECT id::text FROM categories WHERE slug = 'komputer'").Scan(&komputer))
	f.app.createProduct(f.sellerA, productBody("Kabel", komputer, variantBody{"Putih", "AL-9", "20000", 10}))

	_, itemA := f.deliveredItem(t, f.variantA, f.storeA, f.sellerA)
	_, itemB := f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)
	require.Equal(t, http.StatusCreated, f.review(f.buyer, itemA, 3, "").status)
	require.Equal(t, http.StatusCreated, f.review(f.buyer, itemB, 5, "").status)

	res := f.app.do("GET", "/api/v1/products?sort=rating", "", "")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, []string{"Mouse", "Earbuds", "Kabel"}, names(listItems(res)), "best average first, unreviewed last")
	avg, n := productRating(t, f.app.do("GET", "/api/v1/stores/toko-beta/products/mouse", "", ""))
	assert.Equal(t, 5.0, avg)
	assert.Equal(t, 1.0, n)
}

func TestReviews_ConcurrentReviewsKeepTheCountersExact(t *testing.T) {
	f := newCheckoutFixture(t)
	const n = 20
	items := make([]string, n)
	for i := range n {
		_, items[i] = f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)
	}

	// 20 reviews of the same product at once. Each one adds to the counters
	// with an atomic UPDATE, which takes the product row lock, so none is lost.
	statuses := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i] = f.review(f.buyer, items[i], i%5+1, "").status
		}()
	}
	wg.Wait()

	for i, s := range statuses {
		assert.Equal(t, http.StatusCreated, s, "review %d", i)
	}
	assert.Equal(t, n, f.app.count("SELECT rating_count FROM products WHERE id = $1", f.productB))
	assert.Equal(t, 60, f.app.count("SELECT rating_total FROM products WHERE id = $1", f.productB), "4 × (1+2+3+4+5)")
	assert.Equal(t, 0, f.app.ratingMismatches())
}

func TestReviews_SameItemTwiceAtOnce(t *testing.T) {
	f := newCheckoutFixture(t)
	for round := range 5 {
		_, item := f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)
		var wg sync.WaitGroup
		statuses := make([]int, 2)
		for i := range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				statuses[i] = f.review(f.buyer, item, 5, "").status
			}()
		}
		wg.Wait()
		assert.ElementsMatch(t, []int{http.StatusCreated, http.StatusConflict}, statuses, "round %d", round)
	}
	assert.Equal(t, 5, f.app.count("SELECT count(*) FROM reviews"))
	assert.Equal(t, 5, f.app.count("SELECT rating_count FROM products WHERE id = $1", f.productB))
	assert.Equal(t, 0, f.app.ratingMismatches())
}
