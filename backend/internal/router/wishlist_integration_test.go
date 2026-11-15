//go:build integration

// End-to-end tests of the wishlist (practice feature) against marketplace_test.
package router_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f cartFixture) wish(method, session, productID string) result {
	f.app.t.Helper()
	return f.app.do(method, "/api/v1/me/wishlist/"+productID, "", session)
}

func (f cartFixture) wishlistIDs(session string) []any {
	f.app.t.Helper()
	res := f.app.do("GET", "/api/v1/me/wishlist/ids", "", session)
	require.Equal(f.app.t, http.StatusOK, res.status, res.raw)
	ids, _ := res.data()["productIds"].([]any)
	return ids
}

func TestWishlist_RequiresLogin(t *testing.T) {
	f := newCartFixture(t)
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/v1/me/wishlist"},
		{"GET", "/api/v1/me/wishlist/ids"},
		{"PUT", "/api/v1/me/wishlist/" + f.productA},
		{"DELETE", "/api/v1/me/wishlist/" + f.productA},
	} {
		assert.Equal(t, http.StatusUnauthorized, f.app.do(c.method, c.path, "", "").status, c.method+" "+c.path)
	}
}

func TestWishlist_AddListRemove(t *testing.T) {
	f := newCartFixture(t)
	assert.Empty(t, listItems(f.app.do("GET", "/api/v1/me/wishlist", "", f.buyer)), "empty list, not an error")

	require.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productA).status)
	require.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productB).status)

	list := f.app.do("GET", "/api/v1/me/wishlist", "", f.buyer)
	require.Equal(t, http.StatusOK, list.status, list.raw)
	assert.Equal(t, []string{"Mouse", "Earbuds"}, names(listItems(list)), "newest first")
	first := listItems(list)[0].(map[string]any)
	assert.Equal(t, f.productB, first["productId"])
	assert.Equal(t, "50000.00", first["lowestPrice"], "money as a decimal string")
	assert.Equal(t, map[string]any{"slug": "toko-beta", "name": "Toko Beta"}, first["store"])
	assert.Nil(t, first["image"], "no image -> null")
	assert.ElementsMatch(t, []any{f.productA, f.productB}, f.wishlistIDs(f.buyer))

	require.Equal(t, http.StatusNoContent, f.wish("DELETE", f.buyer, f.productA).status)
	assert.Equal(t, []any{f.productB}, f.wishlistIDs(f.buyer))
	assert.Equal(t, http.StatusNoContent, f.wish("DELETE", f.buyer, f.productA).status, "removing again is not an error")
}

func TestWishlist_AddingTwiceKeepsOneRow(t *testing.T) {
	f := newCartFixture(t)
	assert.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productA).status)
	assert.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productA).status)
	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM wishlist_items"))
}

func TestWishlist_OnlyPublicProducts(t *testing.T) {
	f := newCartFixture(t)
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "not-a-uuid"} {
		for _, method := range []string{"PUT", "DELETE"} {
			res := f.wish(method, f.buyer, id)
			if method == "DELETE" && id != "not-a-uuid" {
				// A well-formed id that is not in the wishlist: idempotent remove.
				assert.Equal(t, http.StatusNoContent, res.status, method+" "+id)
				continue
			}
			assert.Equal(t, http.StatusNotFound, res.status, method+" "+id)
			assert.Equal(t, "PRODUCT_NOT_FOUND", res.errorCode())
		}
	}

	// Saved first, archived afterwards: it leaves the list and the ids but its
	// row stays (the product may be published again).
	require.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productA).status)
	require.Equal(t, http.StatusNoContent, f.app.do("DELETE", "/api/v1/seller/products/"+f.productA, "", f.sellerA).status)
	assert.Empty(t, listItems(f.app.do("GET", "/api/v1/me/wishlist", "", f.buyer)))
	assert.Empty(t, f.wishlistIDs(f.buyer))
	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM wishlist_items"))

	// An archived product cannot be added.
	other := f.app.register("Orang Lain", "lain@example.com", "rahasia-123")
	res := f.wish("PUT", other, f.productA)
	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, "PRODUCT_NOT_FOUND", res.errorCode())
}

func TestWishlist_EachUserSeesOnlyTheirOwn(t *testing.T) {
	f := newCartFixture(t)
	other := f.app.register("Orang Lain", "lain@example.com", "rahasia-123")
	require.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productA).status)

	assert.Empty(t, listItems(f.app.do("GET", "/api/v1/me/wishlist", "", other)))
	assert.Empty(t, f.wishlistIDs(other))
	// The other user "removing" the same product only touches their own (empty) list.
	assert.Equal(t, http.StatusNoContent, f.wish("DELETE", other, f.productA).status)
	assert.Equal(t, []any{f.productA}, f.wishlistIDs(f.buyer), "the buyer's item is still there")
}

func TestWishlist_ConcurrentAddsLeaveOneRow(t *testing.T) {
	f := newCartFixture(t)
	// 10 "add" clicks at once. Each INSERT … ON CONFLICT DO NOTHING waits on
	// the unique index for the first one and then does nothing.
	statuses := make([]int, 10)
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i] = f.wish("PUT", f.buyer, f.productB).status
		}()
	}
	wg.Wait()
	for i, s := range statuses {
		assert.Equal(t, http.StatusNoContent, s, "request %d", i)
	}
	assert.Equal(t, 1, f.app.count("SELECT count(*) FROM wishlist_items"))
}

func TestWishlist_Pagination(t *testing.T) {
	f := newCartFixture(t)
	require.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productA).status)
	require.Equal(t, http.StatusNoContent, f.wish("PUT", f.buyer, f.productB).status)

	page2 := f.app.do("GET", "/api/v1/me/wishlist?pageSize=1&page=2", "", f.buyer)
	require.Equal(t, http.StatusOK, page2.status, page2.raw)
	assert.Equal(t, []string{"Earbuds"}, names(listItems(page2)))
	assert.Equal(t, 2.0, page2.data()["total"])
	assert.Equal(t, 2.0, page2.data()["totalPages"])

	past := f.app.do("GET", "/api/v1/me/wishlist?pageSize=1&page=9", "", f.buyer)
	assert.Empty(t, listItems(past))
	assert.Equal(t, 2.0, past.data()["total"], "total is still reported past the last page")

	// 0 means "default" (as in the other list endpoints), so it is not here.
	for _, q := range []string{"pageSize=-1", "pageSize=49", "page=-1", "page=abc"} {
		assert.Equal(t, http.StatusUnprocessableEntity, f.app.do("GET", "/api/v1/me/wishlist?"+q, "", f.buyer).status, q)
	}
}
