//go:build integration

package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addressBody is a valid create body; extra key/value pairs override fields.
func addressBody(label string, extra ...string) string {
	body := map[string]string{
		"label": label, "recipientName": "Budi Santoso", "phone": "0812-3456 7890",
		"addressLine": "Jl. Melati No. 5", "city": "Bandung", "province": "Jawa Barat", "postalCode": "40115",
	}
	for i := 0; i+1 < len(extra); i += 2 {
		body[extra[i]] = extra[i+1]
	}
	b, _ := json.Marshal(body)
	return string(b)
}

func (a *testApp) addAddress(cookie, label string, makeDefault bool) map[string]any {
	a.t.Helper()
	body := map[string]any{
		"label": label, "recipientName": "Budi Santoso", "phone": "081234567890",
		"addressLine": "Jl. Melati No. 5", "city": "Bandung", "province": "Jawa Barat", "postalCode": "40115",
		"isDefault": makeDefault,
	}
	res := a.do("POST", "/api/v1/me/addresses", toJSON(a.t, body), cookie)
	require.Equal(a.t, http.StatusCreated, res.status, res.raw)
	return res.data()
}

func (a *testApp) addresses(cookie string) []map[string]any {
	a.t.Helper()
	res := a.do("GET", "/api/v1/me/addresses", "", cookie)
	require.Equal(a.t, http.StatusOK, res.status, res.raw)
	raw, _ := res.body["data"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any))
	}
	return out
}

func labels(list []map[string]any) []string {
	out := make([]string, 0, len(list))
	for _, a := range list {
		out = append(out, a["label"].(string))
	}
	return out
}

func TestAddresses_RequireLogin(t *testing.T) {
	app := newTestApp(t, generousLimits)
	assert.Equal(t, http.StatusUnauthorized, app.do("GET", "/api/v1/me/addresses", "", "").status)
	assert.Equal(t, http.StatusUnauthorized, app.do("POST", "/api/v1/me/addresses", addressBody("Rumah"), "").status)
}

func TestAddresses_DefaultRules(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")

	assert.Empty(t, app.addresses(buyer), "a new user has no addresses (empty list, not an error)")

	home := app.addAddress(buyer, "Rumah", false)
	assert.Equal(t, true, home["isDefault"], "the first address is always the default")

	office := app.addAddress(buyer, "Kantor", false)
	assert.Equal(t, false, office["isDefault"])

	parents := app.addAddress(buyer, "Orang tua", true)
	assert.Equal(t, true, parents["isDefault"], "isDefault=true moves the default")
	assert.Equal(t, []string{"Orang tua", "Kantor", "Rumah"}, labels(app.addresses(buyer)), "default first, then newest")

	res := app.do("POST", "/api/v1/me/addresses/"+office["id"].(string)+"/default", "", buyer)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, true, res.data()["isDefault"])
	assert.Equal(t, "Kantor", app.addresses(buyer)[0]["label"])
	assert.Equal(t, 1, app.count("SELECT count(*) FROM user_addresses WHERE is_default"))

	// Setting the current default again is a harmless no-op.
	require.Equal(t, http.StatusOK, app.do("POST", "/api/v1/me/addresses/"+office["id"].(string)+"/default", "", buyer).status)
	assert.Equal(t, 1, app.count("SELECT count(*) FROM user_addresses WHERE is_default"))
}

func TestAddresses_ValidationAndNormalisation(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")

	res := app.do("POST", "/api/v1/me/addresses", `{"phone":"12ab","postalCode":"4011"}`, buyer)
	require.Equal(t, http.StatusUnprocessableEntity, res.status, res.raw)
	details, _ := res.body["error"].(map[string]any)["details"].(map[string]any)
	for _, k := range []string{"label", "recipientName", "addressLine", "city", "province", "phone", "postalCode"} {
		assert.Contains(t, details, k)
	}

	created := app.do("POST", "/api/v1/me/addresses", addressBody("  Rumah  "), buyer)
	require.Equal(t, http.StatusCreated, created.status, created.raw)
	assert.Equal(t, "Rumah", created.data()["label"], "trimmed")
	assert.Equal(t, "081234567890", created.data()["phone"], "separators removed")

	// Strict JSON: an unknown field (here: userId, an attempt to pick the owner) is rejected.
	res = app.do("POST", "/api/v1/me/addresses", `{"userId":"00000000-0000-0000-0000-000000000000"}`, buyer)
	assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
}

func TestAddresses_PartialUpdate(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")
	a := app.addAddress(buyer, "Rumah", false)
	path := "/api/v1/me/addresses/" + a["id"].(string)

	res := app.do("PATCH", path, `{"city":"Jakarta Selatan","postalCode":"12160"}`, buyer)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, "Jakarta Selatan", res.data()["city"])
	assert.Equal(t, "12160", res.data()["postalCode"])
	assert.Equal(t, "Rumah", res.data()["label"], "omitted fields stay unchanged")
	assert.Equal(t, true, res.data()["isDefault"])

	assert.Equal(t, http.StatusUnprocessableEntity, app.do("PATCH", path, `{"postalCode":"x"}`, buyer).status)
	assert.Equal(t, http.StatusBadRequest, app.do("PATCH", path, `{"isDefault":false}`, buyer).status,
		"the default flag is not editable through PATCH, so the only default cannot be removed")
}

func TestAddresses_DeletingTheDefaultPromotesTheNewest(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")
	home := app.addAddress(buyer, "Rumah", false) // default
	app.addAddress(buyer, "Kantor", false)
	last := app.addAddress(buyer, "Kos", false)

	require.Equal(t, http.StatusNoContent, app.do("DELETE", "/api/v1/me/addresses/"+home["id"].(string), "", buyer).status)
	list := app.addresses(buyer)
	require.Len(t, list, 2)
	assert.Equal(t, last["id"], list[0]["id"], "the newest remaining address is the new default")
	assert.Equal(t, true, list[0]["isDefault"])

	for _, a := range list {
		require.Equal(t, http.StatusNoContent, app.do("DELETE", "/api/v1/me/addresses/"+a["id"].(string), "", buyer).status)
	}
	assert.Empty(t, app.addresses(buyer))
	assert.Equal(t, http.StatusNotFound, app.do("DELETE", "/api/v1/me/addresses/"+home["id"].(string), "", buyer).status)
}

func TestAddresses_LimitPerUser(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")
	for i := range 10 {
		app.addAddress(buyer, fmt.Sprintf("Alamat %d", i), false)
	}
	res := app.do("POST", "/api/v1/me/addresses", addressBody("Satu lagi"), buyer)
	assert.Equal(t, http.StatusConflict, res.status, res.raw)
	assert.Equal(t, "ADDRESS_LIMIT_REACHED", res.errorCode())
}

func TestAddresses_OtherUsersGetNotFound(t *testing.T) {
	app := newTestApp(t, generousLimits)
	alice := app.register("Alice", "alice@example.com", "rahasia-123")
	bob := app.register("Bob", "bob@example.com", "rahasia-123")
	a := app.addAddress(alice, "Rumah", false)
	path := "/api/v1/me/addresses/" + a["id"].(string)

	// Bob knows Alice's address id. Every route must behave as if it did not exist.
	for _, r := range []struct{ method, path, body string }{
		{"PATCH", path, `{"city":"Surabaya"}`},
		{"DELETE", path, ""},
		{"POST", path + "/default", ""},
	} {
		res := app.do(r.method, r.path, r.body, bob)
		assert.Equal(t, http.StatusNotFound, res.status, "%s %s: %s", r.method, r.path, res.raw)
		assert.Equal(t, "ADDRESS_NOT_FOUND", res.errorCode())
	}
	assert.Empty(t, app.addresses(bob))
	assert.Equal(t, "Bandung", app.addresses(alice)[0]["city"], "Alice's address is untouched")

	assert.Equal(t, http.StatusNotFound, app.do("DELETE", "/api/v1/me/addresses/not-a-uuid", "", alice).status)
}

func TestAddresses_DatabaseAllowsOnlyOneDefault(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")
	app.addAddress(buyer, "Rumah", false) // default

	// Bypass the service: the partial unique index must still refuse a second default.
	_, err := app.pool.Exec(context.Background(), `
		INSERT INTO user_addresses (user_id, label, recipient_name, phone, address_line, city, province, postal_code, is_default)
		SELECT id, 'Kedua', 'Budi', '081234567890', 'Jl. Mawar 1', 'Bandung', 'Jawa Barat', '40115', TRUE
		FROM users WHERE email = 'buyer@example.com'`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_addresses_one_default_per_user_idx")

	// And the CHECK constraints refuse a malformed phone.
	_, err = app.pool.Exec(context.Background(), `
		INSERT INTO user_addresses (user_id, label, recipient_name, phone, address_line, city, province, postal_code)
		SELECT id, 'X', 'Budi', '08-12', 'Jl. Mawar 1', 'Bandung', 'Jawa Barat', '40115'
		FROM users WHERE email = 'buyer@example.com'`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_addresses_phone_format_check")
}

func TestAddresses_ConcurrentCreatesRespectTheLimit(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")
	for i := range 7 {
		app.addAddress(buyer, fmt.Sprintf("Alamat %d", i), false)
	}

	// 6 simultaneous creates with 3 free slots, all asking to become the default.
	// The per-user lock serialises them: exactly 3 succeed, the rest get 409,
	// and there is still exactly one default.
	statuses := make(chan int, 6)
	var wg sync.WaitGroup
	for i := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := toJSON(t, map[string]any{
				"label": fmt.Sprintf("Baru %d", i), "recipientName": "Budi", "phone": "081234567890",
				"addressLine": "Jl. Melati No. 5", "city": "Bandung", "province": "Jawa Barat", "postalCode": "40115",
				"isDefault": true,
			})
			statuses <- app.do("POST", "/api/v1/me/addresses", body, buyer).status
		}()
	}
	wg.Wait()
	close(statuses)

	got := map[int]int{}
	for s := range statuses {
		got[s]++
	}
	assert.Equal(t, map[int]int{http.StatusCreated: 3, http.StatusConflict: 3}, got)
	assert.Equal(t, 10, app.count("SELECT count(*) FROM user_addresses"))
	assert.Equal(t, 1, app.count("SELECT count(*) FROM user_addresses WHERE is_default"))
}

func TestAddresses_ConcurrentDefaultSwitches(t *testing.T) {
	app := newTestApp(t, generousLimits)
	buyer := app.register("Buyer", "buyer@example.com", "rahasia-123")
	var ids []string
	for i := range 5 {
		ids = append(ids, app.addAddress(buyer, fmt.Sprintf("Alamat %d", i), false)["id"].(string))
	}

	// Without the lock, two "clear default, then set" sequences can interleave
	// and the second SET hits the unique index (a 500). With it, all succeed.
	statuses := make(chan int, len(ids))
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- app.do("POST", "/api/v1/me/addresses/"+id+"/default", "", buyer).status
		}()
	}
	wg.Wait()
	close(statuses)
	for s := range statuses {
		assert.Equal(t, http.StatusOK, s)
	}
	assert.Equal(t, 1, app.count("SELECT count(*) FROM user_addresses WHERE is_default"))
}
