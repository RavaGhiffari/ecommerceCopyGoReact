//go:build integration

// Pagination limits on every list endpoint (7d security pass). The offset is
// computed as int32((page-1)*pageSize); without an upper bound on page, a huge
// page overflows into a negative OFFSET and PostgreSQL fails the query, so a
// single crafted URL produced a 500 (and an error log line) instead of a 422.
package router_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveryListEndpoint_RejectsOutOfRangePaging(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Toko Halaman")
	p := app.createProduct(seller, productBody("Topi", c.audio, variantBody{"Hitam", "PG-1", "1000", 1}))
	buyer := app.register("Pembeli", "pembeli@example.com", "rahasia-123")
	admin := app.newAdmin()

	endpoints := []struct {
		path, cookie string
	}{
		{"/api/v1/products", ""},
		{fmt.Sprintf("/api/v1/products/%s/reviews", p["id"]), ""},
		{"/api/v1/me/wishlist", buyer},
		{"/api/v1/me/reviews", buyer},
		{"/api/v1/me/reviews/pending", buyer},
		{"/api/v1/orders", buyer},
		{"/api/v1/seller/orders", seller},
		{"/api/v1/admin/users", admin},
		{"/api/v1/admin/stores", admin},
		{"/api/v1/admin/orders", admin},
		{"/api/v1/admin/reviews", admin},
		{"/api/v1/admin/audit-logs", admin},
	}
	for _, e := range endpoints {
		t.Run(e.path, func(t *testing.T) {
			// 50 000 000 × 50 does not fit in int32: this was the 500.
			for _, q := range []string{"page=50000000&pageSize=48", "page=50000000", "page=10001", "page=-1", "pageSize=-1", "pageSize=1000", "page=abc"} {
				res := app.do("GET", e.path+"?"+q, "", e.cookie)
				assert.Equal(t, http.StatusUnprocessableEntity, res.status, "%s: %s", q, res.raw)
			}
			// The last allowed page is fine, just empty.
			res := app.do("GET", e.path+"?page=6000", "", e.cookie)
			assert.Equal(t, http.StatusOK, res.status, res.raw)
		})
	}
}
