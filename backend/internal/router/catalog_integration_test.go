//go:build integration

// End-to-end tests of Phase 2: categories, stores, products, variants and
// stock, against the real marketplace_test database.
package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/auth"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
)

// ---- helpers ----------------------------------------------------------------

func toJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func (a *testApp) login(email, pw string) string {
	a.t.Helper()
	res := a.do("POST", "/api/v1/auth/login", toJSON(a.t, map[string]string{"email": email, "password": pw}), "")
	require.Equal(a.t, http.StatusOK, res.status, res.raw)
	return res.session().Value
}

// newSeller registers a customer and opens a store; returns their session.
func (a *testApp) newSeller(storeName string) string {
	a.t.Helper()
	email := strings.ReplaceAll(strings.ToLower(storeName), " ", "") + "@example.com"
	cookie := a.register(storeName+" Owner", email, "rahasia-123")
	res := a.do("POST", "/api/v1/seller/store", toJSON(a.t, map[string]string{"name": storeName, "location": "Bandung"}), cookie)
	require.Equal(a.t, http.StatusCreated, res.status, res.raw)
	return cookie
}

func (a *testApp) newAdmin() string {
	a.t.Helper()
	_, err := a.auth.CreateUserWithRole(context.Background(),
		auth.RegisterInput{Name: "Admin", Email: "admin@example.com", Password: "rahasia-admin"}, authctx.RoleAdmin)
	require.NoError(a.t, err)
	return a.login("admin@example.com", "rahasia-admin")
}

type testCategories struct{ elektronik, audio, komputer string }

// seedCategories inserts a small tree directly (admin API is tested separately).
func (a *testApp) seedCategories() testCategories {
	a.t.Helper()
	var c testCategories
	ctx := context.Background()
	require.NoError(a.t, a.pool.QueryRow(ctx, "INSERT INTO categories (name, slug) VALUES ('Elektronik','elektronik') RETURNING id::text").Scan(&c.elektronik))
	require.NoError(a.t, a.pool.QueryRow(ctx, "INSERT INTO categories (parent_id, name, slug) VALUES ($1,'Audio','audio') RETURNING id::text", c.elektronik).Scan(&c.audio))
	require.NoError(a.t, a.pool.QueryRow(ctx, "INSERT INTO categories (parent_id, name, slug) VALUES ($1,'Komputer','komputer') RETURNING id::text", c.elektronik).Scan(&c.komputer))
	return c
}

type variantBody struct {
	Name  string `json:"name"`
	SKU   string `json:"sku"`
	Price any    `json:"price"`
	Stock int    `json:"stock"`
}

func productBody(name, categoryID string, variants ...variantBody) map[string]any {
	return map[string]any{"name": name, "categoryId": categoryID, "description": "Deskripsi " + name, "variants": variants}
}

func (a *testApp) createProduct(cookie string, body map[string]any) map[string]any {
	a.t.Helper()
	res := a.do("POST", "/api/v1/seller/products", toJSON(a.t, body), cookie)
	require.Equal(a.t, http.StatusCreated, res.status, res.raw)
	return res.data()
}

func listItems(r result) []any {
	d := r.data()
	items, _ := d["items"].([]any)
	return items
}

func names(items []any) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.(map[string]any)["name"].(string)
	}
	return out
}

// ---- stores & roles ---------------------------------------------------------

func TestOpenStore_MakesTheUserASeller(t *testing.T) {
	app := newTestApp(t, generousLimits)
	cookie := app.register("Budi", "budi@example.com", "rahasia-123")

	assert.Equal(t, http.StatusForbidden, app.do("GET", "/api/v1/seller/products", "", cookie).status, "a customer is not a seller yet")

	res := app.do("POST", "/api/v1/seller/store", `{"name":"Toko Budi Jaya","location":"Bandung","description":"Serba ada"}`, cookie)
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	assert.Equal(t, "toko-budi-jaya", res.data()["slug"], "slug derived from the name")

	me := app.do("GET", "/api/v1/me", "", cookie)
	assert.ElementsMatch(t, []any{"customer", "seller"}, me.data()["roles"], "role granted in the same transaction")
	assert.Equal(t, http.StatusOK, app.do("GET", "/api/v1/seller/products", "", cookie).status, "same session, new role applies immediately")

	again := app.do("POST", "/api/v1/seller/store", `{"name":"Toko Kedua","location":"Bandung"}`, cookie)
	assert.Equal(t, http.StatusConflict, again.status)
	assert.Equal(t, "STORE_EXISTS", again.errorCode(), "one store per seller")
}

func TestOpenStore_SlugIsUnique(t *testing.T) {
	app := newTestApp(t, generousLimits)
	app.newSeller("Toko Sama")
	other := app.register("Lain", "lain@example.com", "rahasia-123")

	res := app.do("POST", "/api/v1/seller/store", `{"name":"Toko Sama","location":"Bandung"}`, other)
	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "SLUG_TAKEN", res.errorCode())
	assert.Equal(t, 1, app.count("SELECT count(*) FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE r.name = 'seller'"),
		"the failed store must not leave a seller role behind (transaction rolled back)")
}

// ---- categories -------------------------------------------------------------

func TestAdminCategories(t *testing.T) {
	app := newTestApp(t, generousLimits)
	customer := app.register("Budi", "budi@example.com", "rahasia-123")
	admin := app.newAdmin()

	assert.Equal(t, http.StatusForbidden, app.do("POST", "/api/v1/admin/categories", `{"name":"X"}`, customer).status)

	parent := app.do("POST", "/api/v1/admin/categories", `{"name":"Olahraga"}`, admin)
	require.Equal(t, http.StatusCreated, parent.status, parent.raw)
	parentID := parent.data()["id"].(string)

	child := app.do("POST", "/api/v1/admin/categories", fmt.Sprintf(`{"name":"Lari","parentId":%q}`, parentID), admin)
	require.Equal(t, http.StatusCreated, child.status, child.raw)
	childID := child.data()["id"].(string)

	tooDeep := app.do("POST", "/api/v1/admin/categories", fmt.Sprintf(`{"name":"Sepatu Lari","parentId":%q}`, childID), admin)
	assert.Equal(t, http.StatusUnprocessableEntity, tooDeep.status, "only two levels")

	dup := app.do("POST", "/api/v1/admin/categories", `{"name":"Olahraga"}`, admin)
	assert.Equal(t, http.StatusConflict, dup.status)

	// A parent with children cannot be moved under another category.
	other := app.do("POST", "/api/v1/admin/categories", `{"name":"Hobi"}`, admin).data()["id"].(string)
	move := app.do("PUT", "/api/v1/admin/categories/"+parentID, fmt.Sprintf(`{"name":"Olahraga","parentId":%q}`, other), admin)
	assert.Equal(t, http.StatusUnprocessableEntity, move.status)

	list := app.do("GET", "/api/v1/categories", "", "")
	assert.Len(t, list.body["data"], 3)
}

func TestDatabaseTriggerBlocksThirdLevelEvenWithoutTheService(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	_, err := app.pool.Exec(context.Background(), "INSERT INTO categories (parent_id, name, slug) VALUES ($1, 'Headphone', 'headphone')", c.audio)
	assert.ErrorContains(t, err, "more than two levels")
}

// ---- products: create, ownership, uniqueness -------------------------------

func TestCreateProduct_IsPublicWithVariantsStockAndLedger(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Gadget Jaya")

	body := productBody("Earbuds Pro", c.audio,
		variantBody{"Hitam", "gj-01", "349000", 40},
		variantBody{"Putih", "GJ-02", 359000.5, 0}, // a JSON number also works
	)
	body["images"] = []map[string]string{{"url": "https://cdn.example.com/earbuds.jpg", "alt": "Earbuds"}}
	p := app.createProduct(seller, body)

	assert.Equal(t, "earbuds-pro", p["slug"])
	variants := p["variants"].([]any)
	require.Len(t, variants, 2)
	first := variants[0].(map[string]any)
	assert.Equal(t, "GJ-01", first["sku"], "SKU normalised to upper case")
	assert.Equal(t, "349000.00", first["price"], "money is a decimal string")
	assert.Equal(t, float64(40), first["stock"])

	// One ledger row for the initial stock (the 0-stock variant needs none), with the seller as actor.
	assert.Equal(t, 1, app.count("SELECT count(*) FROM inventory_movements WHERE available_delta = 40 AND actor_id IS NOT NULL"))
	assert.Equal(t, 2, app.count("SELECT count(*) FROM inventory"))

	list := app.do("GET", "/api/v1/products", "", "")
	require.Equal(t, http.StatusOK, list.status)
	items := listItems(list)
	require.Len(t, items, 1)
	card := items[0].(map[string]any)
	assert.Equal(t, "349000.00", card["lowestPrice"])
	assert.Equal(t, "359000.50", card["highestPrice"])
	assert.Equal(t, true, card["inStock"])
	assert.Equal(t, "https://cdn.example.com/earbuds.jpg", card["images"].([]any)[0].(map[string]any)["url"])

	detail := app.do("GET", "/api/v1/stores/gadget-jaya/products/earbuds-pro", "", "")
	require.Equal(t, http.StatusOK, detail.status, detail.raw)
	assert.Equal(t, "Audio", detail.data()["category"].(map[string]any)["name"])
}

func TestCreateProduct_Validation(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Gadget Jaya")

	tests := []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"no variants", productBody("Earbuds", c.audio), "variants"},
		{"parent category", productBody("Earbuds", c.elektronik, variantBody{"A", "A1", "1000", 1}), "categoryId"},
		{"negative price", productBody("Earbuds", c.audio, variantBody{"A", "A1", "-1", 1}), "variants[0].price"},
		{"three decimals", productBody("Earbuds", c.audio, variantBody{"A", "A1", "1.234", 1}), "variants[0].price"},
		{"negative stock", productBody("Earbuds", c.audio, variantBody{"A", "A1", "1000", -5}), "variants[0].stock"},
		{"duplicate SKU in one request", productBody("Earbuds", c.audio, variantBody{"A", "X1", "1000", 1}, variantBody{"B", "x1", "1000", 1}), "variants[1].sku"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := app.do("POST", "/api/v1/seller/products", toJSON(t, tt.body), seller)
			require.Equal(t, http.StatusUnprocessableEntity, res.status, res.raw)
			assert.Contains(t, res.body["error"].(map[string]any)["details"], tt.field)
		})
	}

	img := productBody("Earbuds", c.audio, variantBody{"A", "A1", "1000", 1})
	img["images"] = []map[string]string{{"url": "javascript:alert(1)"}}
	res := app.do("POST", "/api/v1/seller/products", toJSON(t, img), seller)
	assert.Equal(t, http.StatusUnprocessableEntity, res.status, "only http(s) image URLs")

	assert.Equal(t, 0, app.count("SELECT count(*) FROM products"), "nothing half-created")
}

func TestSellerCannotTouchAnotherSellersProduct(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	alice := app.newSeller("Toko Alice")
	bob := app.newSeller("Toko Bob")

	p := app.createProduct(alice, productBody("Kabel USB", c.komputer, variantBody{"1m", "AL-1", "39000", 10}))
	id := p["id"].(string)
	variantID := p["variants"].([]any)[0].(map[string]any)["id"].(string)

	// Bob knows the IDs (they are visible in public URLs/JSON) but owns nothing.
	for _, r := range []struct{ method, path, body string }{
		{"GET", "/api/v1/seller/products/" + id, ""},
		{"PATCH", "/api/v1/seller/products/" + id, `{"name":"Dibajak"}`},
		{"DELETE", "/api/v1/seller/products/" + id, ""},
		{"POST", "/api/v1/seller/products/" + id + "/variants", `{"name":"2m","sku":"BOB-1","price":"1","stock":1}`},
		{"PATCH", "/api/v1/seller/products/" + id + "/variants/" + variantID, `{"price":"1"}`},
		{"PUT", "/api/v1/seller/products/" + id + "/variants/" + variantID + "/stock", `{"quantity":0}`},
	} {
		res := app.do(r.method, r.path, r.body, bob)
		assert.Equal(t, http.StatusNotFound, res.status, "%s %s -> %s", r.method, r.path, res.raw)
	}

	mine := app.do("GET", "/api/v1/seller/products/"+id, "", alice)
	require.Equal(t, http.StatusOK, mine.status)
	assert.Equal(t, "Kabel USB", mine.data()["name"])
	assert.Equal(t, "active", mine.data()["status"])
	assert.Equal(t, float64(10), mine.data()["variants"].([]any)[0].(map[string]any)["stock"])
	assert.Len(t, listItems(app.do("GET", "/api/v1/products", "", "")), 1, "still public")
}

func TestProductSlugUniquePerStore_SKUUniqueEverywhere(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	alice := app.newSeller("Toko Alice")
	bob := app.newSeller("Toko Bob")

	app.createProduct(alice, productBody("Kaos Polos", c.komputer, variantBody{"M", "AL-M", "59000", 1}))

	same := app.do("POST", "/api/v1/seller/products", toJSON(t, productBody("Kaos Polos", c.komputer, variantBody{"L", "AL-L", "59000", 1})), alice)
	assert.Equal(t, http.StatusConflict, same.status)
	assert.Equal(t, "SLUG_TAKEN", same.errorCode())

	other := app.do("POST", "/api/v1/seller/products", toJSON(t, productBody("Kaos Polos", c.komputer, variantBody{"M", "BOB-M", "59000", 1})), bob)
	assert.Equal(t, http.StatusCreated, other.status, "another store may use the same slug")

	sku := app.do("POST", "/api/v1/seller/products", toJSON(t, productBody("Kaos Lain", c.komputer, variantBody{"M", "al-m", "59000", 1})), bob)
	assert.Equal(t, http.StatusConflict, sku.status)
	assert.Equal(t, "SKU_TAKEN", sku.errorCode())
	assert.Equal(t, 2, app.count("SELECT count(*) FROM products"), "the conflicting product was rolled back with its variant")
}

// ---- stock ------------------------------------------------------------------

func TestSetStock_WritesLedgerAndNeverGoesNegative(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Toko Stok")
	p := app.createProduct(seller, productBody("Mouse", c.komputer, variantBody{"Hitam", "MS-1", "129000", 40}))
	path := fmt.Sprintf("/api/v1/seller/products/%s/variants/%s/stock", p["id"], p["variants"].([]any)[0].(map[string]any)["id"])

	res := app.do("PUT", path, `{"quantity":15,"reason":"stock opname"}`, seller)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, float64(15), res.data()["availableQuantity"])
	assert.Equal(t, 1, app.count("SELECT count(*) FROM inventory_movements WHERE available_delta = -25 AND reason = 'stock opname'"))

	assert.Equal(t, http.StatusUnprocessableEntity, app.do("PUT", path, `{"quantity":-1}`, seller).status)

	_, err := app.pool.Exec(context.Background(), "UPDATE inventory SET available_quantity = -1")
	assert.ErrorContains(t, err, "inventory_available_check", "the database refuses negative stock on its own")
}

func TestConcurrentStockChanges_LedgerStaysConsistent(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Toko Ramai")
	p := app.createProduct(seller, productBody("Mouse", c.komputer, variantBody{"Hitam", "RM-1", "129000", 0}))
	path := fmt.Sprintf("/api/v1/seller/products/%s/variants/%s/stock", p["id"], p["variants"].([]any)[0].(map[string]any)["id"])

	// 20 requests set different quantities at the same time. Without the row
	// lock, two of them could read the same "current" value and record wrong
	// deltas; with it, the deltas always add up to the final quantity.
	var wg sync.WaitGroup
	for i := 1; i <= 20; i++ {
		wg.Add(1)
		go func(q int) {
			defer wg.Done()
			app.do("PUT", path, fmt.Sprintf(`{"quantity":%d}`, q*10), seller)
		}(i)
	}
	wg.Wait()

	var available, ledgerSum int
	require.NoError(t, app.pool.QueryRow(context.Background(),
		`SELECT i.available_quantity, COALESCE((SELECT SUM(available_delta) FROM inventory_movements m WHERE m.variant_id = i.variant_id), 0)
		 FROM inventory i`).Scan(&available, &ledgerSum))
	assert.Equal(t, available, ledgerSum, "the ledger must explain the current stock exactly")
}

// A product may have at most 20 variants. Several "add variant" requests on a
// product that has 19 must leave exactly 20: one 201, the rest 422. Checking
// the count before the insert (outside a lock) let several of them see 19 and
// all insert (regression test for 7d; it failed on that code).
func TestAddVariant_CapHoldsUnderConcurrency(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Toko Varian")
	variants := make([]variantBody, 19)
	for i := range variants {
		variants[i] = variantBody{fmt.Sprintf("V%d", i), fmt.Sprintf("CAP-%02d", i), "1000", 1}
	}
	p := app.createProduct(seller, productBody("Kaos", c.audio, variants...))
	path := fmt.Sprintf("/api/v1/seller/products/%s/variants", p["id"])

	const racers = 10
	statuses := make([]int, racers)
	var wg sync.WaitGroup
	for i := range racers {
		wg.Go(func() {
			statuses[i] = app.do("POST", path, fmt.Sprintf(`{"name":"N%d","sku":"CAP-N%d","price":"1000","stock":1}`, i, i), seller).status
		})
	}
	wg.Wait()

	created, refused := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusUnprocessableEntity:
			refused++
		}
	}
	assert.Equal(t, 1, created, "statuses: %v", statuses)
	assert.Equal(t, racers-1, refused, "statuses: %v", statuses)
	assert.Equal(t, 20, app.count("SELECT count(*) FROM product_variants WHERE product_id = $1", p["id"]))
}

// ---- public catalog ---------------------------------------------------------

func TestPublicListing_SortSearchFilterPaginate(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Toko Lengkap")
	app.createProduct(seller, productBody("Kabel USB-C Braided", c.komputer, variantBody{"1m", "S-1", "39000", 5}))
	app.createProduct(seller, productBody("Keyboard Mekanikal", c.komputer, variantBody{"Brown", "S-2", "499000", 5}))
	app.createProduct(seller, productBody("Earbuds Bluetooth", c.audio, variantBody{"Hitam", "S-3", "349000", 0}))

	asc := app.do("GET", "/api/v1/products?sort=price_asc", "", "")
	assert.Equal(t, []string{"Kabel USB-C Braided", "Earbuds Bluetooth", "Keyboard Mekanikal"}, names(listItems(asc)))
	desc := app.do("GET", "/api/v1/products?sort=price_desc", "", "")
	assert.Equal(t, "Keyboard Mekanikal", names(listItems(desc))[0])

	assert.Equal(t, []string{"Keyboard Mekanikal"}, names(listItems(app.do("GET", "/api/v1/products?q=keyb", "", ""))), "prefix search")
	assert.Empty(t, listItems(app.do("GET", "/api/v1/products?q=kabel+keyboard", "", "")), "all words must match")
	assert.Len(t, listItems(app.do("GET", "/api/v1/products?q=%27%3A*%26%7C!(", "", "")), 3, "operator-only input is ignored, not an error")

	assert.Len(t, listItems(app.do("GET", "/api/v1/products?category=elektronik", "", "")), 3, "parent includes subcategories")
	assert.Len(t, listItems(app.do("GET", "/api/v1/products?category=audio", "", "")), 1)
	assert.Equal(t, http.StatusNotFound, app.do("GET", "/api/v1/products?category=nope", "", "").status)

	page2 := app.do("GET", "/api/v1/products?sort=price_asc&pageSize=2&page=2", "", "")
	assert.Equal(t, []string{"Keyboard Mekanikal"}, names(listItems(page2)))
	assert.Equal(t, float64(3), page2.data()["total"])
	assert.Equal(t, float64(2), page2.data()["totalPages"])

	past := app.do("GET", "/api/v1/products?pageSize=2&page=9", "", "")
	assert.Empty(t, listItems(past))
	assert.Equal(t, float64(3), past.data()["total"], "total is still reported past the last page")

	// sort=rating exists since Phase 6a (reviews_integration_test.go); an
	// unknown value is still refused.
	assert.Equal(t, http.StatusUnprocessableEntity, app.do("GET", "/api/v1/products?sort=cheapest", "", "").status)
	assert.Equal(t, http.StatusUnprocessableEntity, app.do("GET", "/api/v1/products?pageSize=1000", "", "").status)
}

func TestArchivedProductsDisappearFromThePublicCatalog(t *testing.T) {
	app := newTestApp(t, generousLimits)
	c := app.seedCategories()
	seller := app.newSeller("Toko Arsip")
	p := app.createProduct(seller, productBody("Kemeja Lama", c.komputer, variantBody{"M", "AR-1", "99000", 3}))

	assert.Equal(t, http.StatusNoContent, app.do("DELETE", "/api/v1/seller/products/"+p["id"].(string), "", seller).status)

	assert.Empty(t, listItems(app.do("GET", "/api/v1/products", "", "")))
	assert.Equal(t, http.StatusNotFound, app.do("GET", "/api/v1/stores/toko-arsip/products/kemeja-lama", "", "").status)
	assert.Equal(t, float64(0), app.do("GET", "/api/v1/stores/toko-arsip", "", "").data()["activeProductCount"])

	mine := app.do("GET", "/api/v1/seller/products", "", seller)
	require.Len(t, mine.body["data"], 1, "the seller still sees it")
	assert.Equal(t, "archived", mine.body["data"].([]any)[0].(map[string]any)["status"])
	assert.Equal(t, 1, app.count("SELECT count(*) FROM products"), "archived, not deleted")
}
