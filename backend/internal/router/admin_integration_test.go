//go:build integration

// End-to-end tests of Phase 6d (admin API) against marketplace_test.
package router_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/audit"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/auth"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
)

// adminFixture: the checkout fixture (two stores, a buyer) plus an admin.
type adminFixture struct {
	checkoutFixture
	admin string // session
}

func newAdminFixture(t *testing.T) adminFixture {
	f := adminFixture{checkoutFixture: newCheckoutFixture(t)}
	f.admin = f.app.newAdmin()
	return f
}

func (a *testApp) userID(email string) string {
	a.t.Helper()
	var id string
	require.NoError(a.t, a.pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&id))
	return id
}

func (a *testApp) auditRows(action string) int {
	return a.count("SELECT count(*) FROM audit_logs WHERE action = $1", action)
}

func (f adminFixture) setUserStatus(session, userID, status string) result {
	f.app.t.Helper()
	return f.app.do("PATCH", "/api/v1/admin/users/"+userID+"/status", `{"status":"`+status+`"}`, session)
}

func (f adminFixture) setStoreStatus(session, storeID, status string) result {
	f.app.t.Helper()
	return f.app.do("PATCH", "/api/v1/admin/stores/"+storeID+"/status", `{"status":"`+status+`"}`, session)
}

func TestAdmin_EveryRouteNeedsTheAdminRole(t *testing.T) {
	f := newAdminFixture(t)
	id := uuid.NewString()
	routes := []struct{ method, path, body string }{
		{"GET", "/api/v1/admin/stats", ""},
		{"GET", "/api/v1/admin/users", ""},
		{"PATCH", "/api/v1/admin/users/" + id + "/status", `{"status":"suspended"}`},
		{"GET", "/api/v1/admin/stores", ""},
		{"PATCH", "/api/v1/admin/stores/" + id + "/status", `{"status":"suspended"}`},
		{"GET", "/api/v1/admin/orders", ""},
		{"GET", "/api/v1/admin/orders/" + id, ""},
		{"GET", "/api/v1/admin/reviews", ""},
		{"PATCH", "/api/v1/admin/reviews/" + id + "/status", `{"status":"hidden"}`},
		{"GET", "/api/v1/admin/audit-logs", ""},
		{"POST", "/api/v1/admin/categories", `{"name":"X"}`},
	}
	for _, r := range routes {
		name := r.method + " " + r.path
		assert.Equal(t, http.StatusUnauthorized, f.app.do(r.method, r.path, r.body, "").status, "anonymous: "+name)
		assert.Equal(t, http.StatusForbidden, f.app.do(r.method, r.path, r.body, f.buyer).status, "customer: "+name)
		assert.Equal(t, http.StatusForbidden, f.app.do(r.method, r.path, r.body, f.sellerA).status, "seller: "+name)
	}
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM audit_logs"))
}

func TestAdmin_SuspendAndReactivateUser(t *testing.T) {
	f := newAdminFixture(t)
	buyerID := f.app.userID("pembeli@example.com")

	// Search and filter.
	found := f.app.do("GET", "/api/v1/admin/users?q=PEMBELI", "", f.admin)
	require.Equal(t, http.StatusOK, found.status, found.raw)
	require.Len(t, listItems(found), 1)
	u := listItems(found)[0].(map[string]any)
	assert.Equal(t, "pembeli@example.com", u["email"])
	assert.Equal(t, []any{"customer"}, u["roles"])
	assert.NotContains(t, found.raw, "password", "never any password field")
	assert.Equal(t, float64(4), f.app.do("GET", "/api/v1/admin/users", "", f.admin).data()["total"], "2 sellers, buyer, admin")
	assert.Len(t, listItems(f.app.do("GET", "/api/v1/admin/users?q=%25", "", f.admin)), 0, "% is matched literally")

	// Suspend: the buyer's session dies at once and login is refused.
	res := f.setUserStatus(f.admin, buyerID, "suspended")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, "suspended", res.data()["status"])
	assert.Equal(t, http.StatusUnauthorized, f.app.do("GET", "/api/v1/me", "", f.buyer).status)
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM user_sessions WHERE user_id = $1", buyerID))
	assert.Equal(t, "ACCOUNT_SUSPENDED", f.app.do("POST", "/api/v1/auth/login", `{"email":"pembeli@example.com","password":"rahasia-123"}`, "").errorCode())
	suspended := f.app.do("GET", "/api/v1/admin/users?status=suspended", "", f.admin)
	require.Len(t, listItems(suspended), 1)

	// Repeating the request changes nothing and writes no second audit row.
	require.Equal(t, http.StatusOK, f.setUserStatus(f.admin, buyerID, "suspended").status)
	assert.Equal(t, 1, f.app.auditRows("user.suspend"))
	assert.Equal(t, 1, f.app.count(`SELECT count(*) FROM audit_logs WHERE action = 'user.suspend'
		AND target_id = $1 AND details->>'email' = 'pembeli@example.com' AND (details->>'sessionsEnded')::int = 1`, buyerID))

	// Reactivate: the old session stays dead, a new login works.
	require.Equal(t, http.StatusOK, f.setUserStatus(f.admin, buyerID, "active").status)
	assert.Equal(t, http.StatusUnauthorized, f.app.do("GET", "/api/v1/me", "", f.buyer).status, "old sessions are not revived")
	again := f.app.login("pembeli@example.com", "rahasia-123")
	assert.Equal(t, http.StatusOK, f.app.do("GET", "/api/v1/me", "", again).status)
	assert.Equal(t, 1, f.app.auditRows("user.reactivate"))
}

func TestAdmin_UserStatusRules(t *testing.T) {
	f := newAdminFixture(t)
	adminID := f.app.userID("admin@example.com")
	_, err := f.app.auth.CreateUserWithRole(context.Background(),
		auth.RegisterInput{Name: "Admin Dua", Email: "admin2@example.com", Password: "rahasia-admin"}, authctx.RoleAdmin)
	require.NoError(t, err)
	otherAdmin := f.app.userID("admin2@example.com")
	buyerID := f.app.userID("pembeli@example.com")

	cases := []struct {
		name, id, body string
		status         int
		code           string
	}{
		{"yourself", adminID, `{"status":"suspended"}`, http.StatusConflict, "CANNOT_CHANGE_SELF"},
		{"another admin", otherAdmin, `{"status":"suspended"}`, http.StatusConflict, "CANNOT_CHANGE_ADMIN"},
		{"unknown user", uuid.NewString(), `{"status":"suspended"}`, http.StatusNotFound, "USER_NOT_FOUND"},
		{"malformed id", "nope", `{"status":"suspended"}`, http.StatusNotFound, "USER_NOT_FOUND"},
		{"bad status", buyerID, `{"status":"banned"}`, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"missing status", buyerID, `{}`, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"unknown field", buyerID, `{"status":"suspended","role":"admin"}`, http.StatusBadRequest, ""},
	}
	for _, c := range cases {
		res := f.app.do("PATCH", "/api/v1/admin/users/"+c.id+"/status", c.body, f.admin)
		assert.Equal(t, c.status, res.status, "%s: %s", c.name, res.raw)
		if c.code != "" {
			assert.Equal(t, c.code, res.errorCode(), c.name)
		}
	}
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM audit_logs"), "refused requests leave no audit row")
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM users WHERE status = 'suspended'"))
}

func TestAdmin_SuspendStore(t *testing.T) {
	f := newAdminFixture(t)
	p := f.paidCheckout(t) // store A has a paid order
	require.Equal(t, http.StatusOK, f.add(f.buyer, f.variantA, 1).status)

	list := f.app.do("GET", "/api/v1/admin/stores?q=alpha", "", f.admin)
	require.Equal(t, http.StatusOK, list.status, list.raw)
	require.Len(t, listItems(list), 1)
	s := listItems(list)[0].(map[string]any)
	assert.Equal(t, "tokoalpha@example.com", s["owner"].(map[string]any)["email"])
	assert.Equal(t, float64(1), s["activeProductCount"])

	res := f.setStoreStatus(f.admin, f.storeA, "suspended")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, "suspended", res.data()["status"])

	// Out of the catalog, flagged in the cart.
	assert.Equal(t, []string{"Mouse"}, names(listItems(f.app.do("GET", "/api/v1/products", "", ""))))
	cart := f.app.do("GET", "/api/v1/cart", "", f.buyer)
	assert.Equal(t, "unavailable", firstItem(cart)["issue"])
	assert.Equal(t, http.StatusNotFound, f.app.do("GET", "/api/v1/stores/toko-alpha", "", "").status)

	// The existing order still moves forward (PLAN.md Q10).
	assert.Equal(t, http.StatusOK, f.setStatus(f.sellerA, p.orderA, `{"status":"processing"}`).status)

	// Reactivate: back in the catalog.
	require.Equal(t, http.StatusOK, f.setStoreStatus(f.admin, f.storeA, "active").status)
	assert.ElementsMatch(t, []string{"Earbuds", "Mouse"}, names(listItems(f.app.do("GET", "/api/v1/products", "", ""))))
	assert.Equal(t, 1, f.app.auditRows("store.suspend"))
	assert.Equal(t, 1, f.app.auditRows("store.reactivate"))
	assert.Equal(t, http.StatusNotFound, f.setStoreStatus(f.admin, uuid.NewString(), "active").status)
}

func TestAdmin_HideAndUnhideReview(t *testing.T) {
	f := newAdminFixture(t)
	_, item1 := f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)
	_, item2 := f.deliveredItem(t, f.variantB, f.storeB, f.sellerB)
	require.Equal(t, http.StatusCreated, f.review(f.buyer, item1, 4, "Bagus").status)
	low := f.review(f.buyer, item2, 2, "Kurang")
	require.Equal(t, http.StatusCreated, low.status)
	lowID := low.data()["id"].(string)
	public := func() map[string]any {
		return f.app.do("GET", "/api/v1/products/"+f.productB+"/reviews", "", "").data()
	}
	assert.Equal(t, map[string]any{"average": float64(3), "count": float64(2)},
		pick(public()["summary"].(map[string]any), "average", "count"))

	// Hide: out of the aggregate and the public list; the author still sees it.
	res := f.app.do("PATCH", "/api/v1/admin/reviews/"+lowID+"/status", `{"status":"hidden"}`, f.admin)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, "hidden", res.data()["status"])
	assert.Equal(t, map[string]any{"average": float64(4), "count": float64(1)},
		pick(public()["summary"].(map[string]any), "average", "count"))
	assert.Len(t, public()["items"], 1)
	assert.Contains(t, f.app.do("GET", "/api/v1/me/reviews", "", f.buyer).raw, `"status":"hidden"`)
	assert.Equal(t, 0, f.app.ratingMismatches())

	hidden := f.app.do("GET", "/api/v1/admin/reviews?status=hidden", "", f.admin)
	require.Len(t, listItems(hidden), 1)
	assert.Equal(t, "pembeli@example.com", listItems(hidden)[0].(map[string]any)["reviewer"].(map[string]any)["email"])

	// Unhide: back, and not marked as edited by its author (migration 000015).
	require.Equal(t, http.StatusOK, f.app.do("PATCH", "/api/v1/admin/reviews/"+lowID+"/status", `{"status":"visible"}`, f.admin).status)
	assert.Equal(t, map[string]any{"average": float64(3), "count": float64(2)},
		pick(public()["summary"].(map[string]any), "average", "count"))
	for _, it := range public()["items"].([]any) {
		assert.Equal(t, false, it.(map[string]any)["edited"])
	}
	assert.Equal(t, 0, f.app.ratingMismatches())
	assert.Equal(t, 1, f.app.auditRows("review.hide"))
	assert.Equal(t, 1, f.app.auditRows("review.unhide"))

	// An author edit while hidden does not touch the counters.
	require.Equal(t, http.StatusOK, f.app.do("PATCH", "/api/v1/admin/reviews/"+lowID+"/status", `{"status":"hidden"}`, f.admin).status)
	require.Equal(t, http.StatusOK, f.app.do("PATCH", "/api/v1/reviews/"+lowID, `{"rating":5}`, f.buyer).status)
	assert.Equal(t, 0, f.app.ratingMismatches())
	assert.Equal(t, http.StatusNotFound, f.app.do("PATCH", "/api/v1/admin/reviews/"+uuid.NewString()+"/status", `{"status":"hidden"}`, f.admin).status)
}

func pick(m map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		out[k] = m[k]
	}
	return out
}

// Hand-computed: store A Earbuds 100000.50, store B Mouse 50000.
//
//	p1 paid      A: 2 × 100000.50 = 200001.00   B: 3 × 50000 = 150000.00 (backdated 40 days)
//	p2 pending   A: 2 units                      B: 3 units                (not GMV)
func TestAdmin_StatsMatchHandComputedNumbers(t *testing.T) {
	f := newAdminFixture(t)
	p1 := f.paidCheckout(t)
	f.placeTwoStores(t)
	f.app.exec("UPDATE orders SET created_at = now() - interval '40 days' WHERE id = $1", p1.orderB)
	require.Equal(t, http.StatusOK, f.setStoreStatus(f.admin, f.storeB, "suspended").status)
	require.Equal(t, http.StatusOK, f.setUserStatus(f.admin, f.app.userID("pembeli@example.com"), "suspended").status)

	res := f.app.do("GET", "/api/v1/admin/stats", "", f.admin)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	d := res.data()
	assert.Equal(t, map[string]any{"active": float64(3), "suspended": float64(1)}, d["users"])
	assert.Equal(t, map[string]any{"active": float64(1), "suspended": float64(1)}, d["stores"])
	assert.Equal(t, float64(1), d["publicProducts"], "store B's Mouse is hidden with its store")
	byStatus := d["ordersByStatus"].(map[string]any)
	assert.Len(t, byStatus, 7)
	assert.Equal(t, float64(2), byStatus["paid"])
	assert.Equal(t, float64(2), byStatus["pending_payment"])
	assert.Equal(t, map[string]any{"last30Days": "200001.00", "allTime": "350001.00"}, d["gmv"])
}

func TestAdmin_PlatformOrders(t *testing.T) {
	f := newAdminFixture(t)
	p := f.paidCheckout(t)
	f.placeTwoStores(t)

	all := f.app.do("GET", "/api/v1/admin/orders", "", f.admin)
	require.Equal(t, http.StatusOK, all.status, all.raw)
	assert.Equal(t, float64(4), all.data()["total"], "both stores, every buyer")
	paid := f.app.do("GET", "/api/v1/admin/orders?status=paid&pageSize=1", "", f.admin)
	assert.Equal(t, float64(2), paid.data()["total"])
	assert.Len(t, listItems(paid), 1)
	assert.Equal(t, http.StatusUnprocessableEntity, f.app.do("GET", "/api/v1/admin/orders?status=lost", "", f.admin).status)

	one := f.app.do("GET", "/api/v1/admin/orders/"+p.orderB, "", f.admin)
	require.Equal(t, http.StatusOK, one.status, one.raw)
	assert.Equal(t, "Mouse", one.data()["items"].([]any)[0].(map[string]any)["productName"])
	assert.Len(t, one.data()["history"], 2)
	assert.Equal(t, "ORDER_NOT_FOUND", f.app.do("GET", "/api/v1/admin/orders/"+uuid.NewString(), "", f.admin).errorCode())
}

func TestAdmin_AuditLog(t *testing.T) {
	f := newAdminFixture(t)
	created := f.app.do("POST", "/api/v1/admin/categories", `{"name":"Mainan"}`, f.admin)
	require.Equal(t, http.StatusCreated, created.status, created.raw)
	catID := created.data()["id"].(string)
	require.Equal(t, http.StatusOK, f.app.do("PUT", "/api/v1/admin/categories/"+catID, `{"name":"Mainan Anak"}`, f.admin).status)
	require.Equal(t, http.StatusOK, f.setStoreStatus(f.admin, f.storeA, "suspended").status)
	// A refused category write leaves no row.
	assert.Equal(t, http.StatusUnprocessableEntity, f.app.do("POST", "/api/v1/admin/categories", `{"name":""}`, f.admin).status)

	res := f.app.do("GET", "/api/v1/admin/audit-logs", "", f.admin)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	items := listItems(res)
	require.Len(t, items, 3)
	var actions []string
	for _, it := range items {
		actions = append(actions, it.(map[string]any)["action"].(string))
	}
	assert.Equal(t, []string{"store.suspend", "category.update", "category.create"}, actions, "newest first")
	upd := items[1].(map[string]any)
	assert.Equal(t, "admin@example.com", upd["actor"].(map[string]any)["email"])
	assert.Equal(t, catID, upd["targetId"])
	assert.Equal(t, "Mainan", upd["details"].(map[string]any)["before"].(map[string]any)["name"])
	assert.Equal(t, "Mainan Anak", upd["details"].(map[string]any)["after"].(map[string]any)["name"])
}

// The audit row and the action share one transaction: when anything after
// audit.Write fails, the row is rolled back with the action.
func TestAdmin_AuditRowRollsBackWithItsAction(t *testing.T) {
	f := newAdminFixture(t)
	adminID := uuid.MustParse(f.app.userID("admin@example.com"))
	buyerID := uuid.MustParse(f.app.userID("pembeli@example.com"))
	boom := errors.New("action failed after the audit row")

	err := database.WithTx(context.Background(), f.app.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(context.Background(), "UPDATE users SET status = 'suspended' WHERE id = $1", buyerID); err != nil {
			return err
		}
		if err := audit.Write(context.Background(), tx, audit.Entry{
			ActorID: adminID, Action: audit.UserSuspend, TargetType: audit.TargetUser, TargetID: buyerID,
		}); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM audit_logs"))
	assert.Equal(t, 0, f.app.count("SELECT count(*) FROM users WHERE status = 'suspended'"))

	// And the table refuses an entry without a real actor.
	err = audit.Write(context.Background(), f.app.pool, audit.Entry{
		ActorID: uuid.New(), Action: audit.UserSuspend, TargetType: audit.TargetUser, TargetID: buyerID,
	})
	assert.ErrorContains(t, err, "audit_logs_actor_id_fkey", fmt.Sprint("actor must exist"))
}
