//go:build integration

// End-to-end tests of the auth flows: real router, real middleware, real
// services, real PostgreSQL (the marketplace_test database; see `make test-db`).
// Each test starts from empty user tables.
package router_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/middleware"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/admin"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/auth"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/cart"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/categories"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/checkout"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/health"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/orders"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/products"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/reviews"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/shipping"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/stores"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/users"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/wishlist"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/password"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/router"
)

var generousLimits = auth.Limits{LoginAttempts: 1000, LoginWindow: time.Minute, RegisterAttempts: 1000, RegisterWindow: time.Minute}

type testApp struct {
	t        *testing.T
	r        *gin.Engine
	pool     *pgxpool.Pool
	auth     *auth.Service
	checkout *checkout.Service // to run the expiry sweep directly
}

func newTestApp(t *testing.T, limits auth.Limits) *testApp {
	t.Helper()
	return newTestAppWith(t, limits, nil)
}

// newTestAppWith lets a test change the config (production mode, trusted
// proxies) before the app is built, the same way main.go builds it.
func newTestAppWith(t *testing.T, limits auth.Limits, adjust func(*config.Config)) *testApp {
	t.Helper()
	gin.SetMode(gin.TestMode)

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL is not set; run `make test-integration`")
	}
	if !strings.Contains(url, "_test") {
		t.Fatal("refusing to run: TEST_DATABASE_URL must point at a *_test database, these tests delete data")
	}

	pool, err := database.NewPool(context.Background(), config.DBConfig{
		URL: url, MaxConns: 4, MinConns: 0, MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute, HealthCheckPeriod: time.Minute, ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// CASCADE also empties every table that references these (user_roles,
	// user_sessions, stores, products, variants, inventory, movements). roles
	// stays: it is schema data created by the migration.
	_, err = pool.Exec(context.Background(), "TRUNCATE users, categories CASCADE")
	require.NoError(t, err)

	authSvc, err := auth.NewService(pool, auth.Config{
		SessionTTL: time.Hour,
		Password:   password.Params{MemoryKiB: 8 * 1024, Iterations: 1, Parallelism: 1}, // fast for tests
	})
	require.NoError(t, err)

	cfg := config.Config{
		AppEnv:             "development",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
		RequestTimeout:     5 * time.Second,
	}
	if adjust != nil {
		adjust(&cfg)
	}
	checkoutSvc := checkout.NewService(pool, checkout.Config{TTL: 30 * time.Minute, Rates: shipping.DefaultRates()})
	h := router.Handlers{
		Health:     health.NewHandler(health.NewRepository(pool)),
		Addresses:  addresses.NewHandler(addresses.NewService(pool)),
		Auth:       auth.NewHandler(authSvc, cfg.SecureCookies(), limits),
		Users:      users.NewHandler(users.NewService(pool)),
		Categories: categories.NewHandler(categories.NewService(pool)),
		Stores:     stores.NewHandler(stores.NewService(pool)),
		Products:   products.NewHandler(products.NewService(pool)),
		Cart:       cart.NewHandler(cart.NewService(pool)),
		Checkout:   checkout.NewHandler(checkoutSvc),
		Orders:     orders.NewHandler(orders.NewService(pool)),
		Reviews:    reviews.NewHandler(reviews.NewService(pool)),
		Wishlist:   wishlist.NewHandler(wishlist.NewService(pool)),
		Admin:      admin.NewHandler(admin.NewService(pool)),
	}
	return &testApp{t: t, r: router.New(cfg, h, authSvc.ResolveSession), pool: pool, auth: authSvc, checkout: checkoutSvc}
}

type result struct {
	status  int
	body    map[string]any
	raw     string
	header  http.Header
	cookies []*http.Cookie
}

func (a *testApp) do(method, path, body, sessionCookie string, headers ...string) result {
	a.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	a.r.ServeHTTP(w, req)

	res := result{status: w.Code, raw: w.Body.String(), header: w.Header(), cookies: w.Result().Cookies()}
	if w.Body.Len() > 0 {
		require.NoError(a.t, json.Unmarshal(w.Body.Bytes(), &res.body), "body: %s", w.Body.String())
	}
	return res
}

func (r result) session() *http.Cookie {
	for _, c := range r.cookies {
		if c.Name == middleware.SessionCookieName {
			return c
		}
	}
	return nil
}

func (r result) errorCode() string {
	e, _ := r.body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func (r result) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (a *testApp) register(name, email, pw string) string {
	a.t.Helper()
	res := a.do("POST", "/api/v1/auth/register",
		`{"name":"`+name+`","email":"`+email+`","password":"`+pw+`"}`, "")
	require.Equal(a.t, http.StatusCreated, res.status, res.raw)
	return res.session().Value
}

func (a *testApp) count(sql string, args ...any) int {
	a.t.Helper()
	var n int
	require.NoError(a.t, a.pool.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}

// ---- register ---------------------------------------------------------------

func TestRegister_CreatesCustomerAndLogsIn(t *testing.T) {
	app := newTestApp(t, generousLimits)

	res := app.do("POST", "/api/v1/auth/register",
		`{"name":"  Rava  ","email":"Rava@Example.com","password":"rahasia-123"}`, "")

	require.Equal(t, http.StatusCreated, res.status, res.raw)
	user := res.data()
	assert.Equal(t, "rava@example.com", user["email"], "stored lowercase")
	assert.Equal(t, "Rava", user["name"], "trimmed")
	assert.Equal(t, []any{"customer"}, user["roles"])
	assert.NotContains(t, res.raw, "password", "no password or hash in any response")

	cookie := res.session()
	require.NotNil(t, cookie)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.InDelta(t, time.Hour.Seconds(), float64(cookie.MaxAge), 5)

	// The database holds only SHA-256(token), never the token itself.
	sum := sha256.Sum256([]byte(cookie.Value))
	assert.Equal(t, 1, app.count("SELECT count(*) FROM user_sessions WHERE token_hash = $1", sum[:]))
	assert.Equal(t, 0, app.count("SELECT count(*) FROM user_sessions WHERE token_hash = $1", []byte(cookie.Value)))
	assert.Equal(t, 1, app.count("SELECT count(*) FROM users WHERE password_hash LIKE '$argon2id$%'"))

	me := app.do("GET", "/api/v1/me", "", cookie.Value)
	require.Equal(t, http.StatusOK, me.status, me.raw)
	assert.Equal(t, "rava@example.com", me.data()["email"])
}

func TestRegister_EmailIsUniqueIgnoringCase(t *testing.T) {
	app := newTestApp(t, generousLimits)
	app.register("A", "rava@example.com", "rahasia-123")

	res := app.do("POST", "/api/v1/auth/register",
		`{"name":"B","email":"RAVA@example.COM","password":"rahasia-456"}`, "")

	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "EMAIL_TAKEN", res.errorCode())
	assert.Equal(t, 1, app.count("SELECT count(*) FROM users"))
}

func TestRegister_ClientCannotChooseRole(t *testing.T) {
	app := newTestApp(t, generousLimits)

	res := app.do("POST", "/api/v1/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"rahasia-123","role":"admin"}`, "")

	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, 0, app.count("SELECT count(*) FROM users"), "nothing created")
}

func TestRegister_ValidationErrorsListEveryField(t *testing.T) {
	app := newTestApp(t, generousLimits)

	res := app.do("POST", "/api/v1/auth/register", `{"name":"","email":"nope","password":"short"}`, "")

	assert.Equal(t, http.StatusUnprocessableEntity, res.status)
	details := res.body["error"].(map[string]any)["details"].(map[string]any)
	assert.Contains(t, details, "name")
	assert.Contains(t, details, "email")
	assert.Contains(t, details, "password")
}

// ---- login / logout / sessions ---------------------------------------------

func TestLogin_UnknownEmailAndWrongPasswordLookIdentical(t *testing.T) {
	app := newTestApp(t, generousLimits)
	app.register("Rava", "rava@example.com", "rahasia-123")

	wrongPassword := app.do("POST", "/api/v1/auth/login", `{"email":"rava@example.com","password":"salah-salah"}`, "")
	unknownEmail := app.do("POST", "/api/v1/auth/login", `{"email":"ghost@example.com","password":"salah-salah"}`, "")

	assert.Equal(t, http.StatusUnauthorized, wrongPassword.status)
	assert.Equal(t, wrongPassword.status, unknownEmail.status)
	assert.Equal(t, wrongPassword.raw, unknownEmail.raw, "same answer, so emails cannot be probed")
	assert.Nil(t, wrongPassword.session())
}

func TestLogin_RotatesTheSession(t *testing.T) {
	app := newTestApp(t, generousLimits)
	oldToken := app.register("Rava", "rava@example.com", "rahasia-123")

	res := app.do("POST", "/api/v1/auth/login", `{"email":"RAVA@example.com","password":"rahasia-123"}`, oldToken)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	newToken := res.session().Value

	assert.NotEqual(t, oldToken, newToken)
	assert.Equal(t, http.StatusUnauthorized, app.do("GET", "/api/v1/me", "", oldToken).status, "old session deleted")
	assert.Equal(t, http.StatusOK, app.do("GET", "/api/v1/me", "", newToken).status)
}

func TestLogout_InvalidatesTheSessionServerSide(t *testing.T) {
	app := newTestApp(t, generousLimits)
	token := app.register("Rava", "rava@example.com", "rahasia-123")

	res := app.do("POST", "/api/v1/auth/logout", "", token)
	assert.Equal(t, http.StatusNoContent, res.status)
	assert.Less(t, res.session().MaxAge, 0, "browser is told to delete the cookie")

	// Even if someone kept a copy of the cookie, it no longer works.
	assert.Equal(t, http.StatusUnauthorized, app.do("GET", "/api/v1/me", "", token).status)
	assert.Equal(t, 0, app.count("SELECT count(*) FROM user_sessions"))
}

func TestExpiredSessionIsRejected(t *testing.T) {
	app := newTestApp(t, generousLimits)
	token := app.register("Rava", "rava@example.com", "rahasia-123")

	_, err := app.pool.Exec(context.Background(),
		"UPDATE user_sessions SET created_at = NOW() - INTERVAL '2 hours', expires_at = NOW() - INTERVAL '1 hour'")
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, app.do("GET", "/api/v1/me", "", token).status)
}

func TestSuspendedUser_IsLoggedOutAndCannotLogIn(t *testing.T) {
	app := newTestApp(t, generousLimits)
	token := app.register("Rava", "rava@example.com", "rahasia-123")

	_, err := app.pool.Exec(context.Background(), "UPDATE users SET status = 'suspended'")
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, app.do("GET", "/api/v1/me", "", token).status, "existing session stops working")

	res := app.do("POST", "/api/v1/auth/login", `{"email":"rava@example.com","password":"rahasia-123"}`, "")
	assert.Equal(t, http.StatusForbidden, res.status)
	assert.Equal(t, "ACCOUNT_SUSPENDED", res.errorCode())

	res = app.do("POST", "/api/v1/auth/login", `{"email":"rava@example.com","password":"wrong-password"}`, "")
	assert.Equal(t, "INVALID_CREDENTIALS", res.errorCode(), "suspension is only revealed after a correct password")
}

func TestLogin_RateLimitedAfterConsecutiveFailures(t *testing.T) {
	app := newTestApp(t, auth.Limits{LoginAttempts: 2, LoginWindow: time.Minute, RegisterAttempts: 100, RegisterWindow: time.Minute})
	app.register("Rava", "rava@example.com", "rahasia-123")
	wrong := `{"email":"rava@example.com","password":"wrong-password"}`
	right := `{"email":"rava@example.com","password":"rahasia-123"}`

	// A success in between resets the counter.
	assert.Equal(t, http.StatusUnauthorized, app.do("POST", "/api/v1/auth/login", wrong, "").status)
	assert.Equal(t, http.StatusOK, app.do("POST", "/api/v1/auth/login", right, "").status)

	assert.Equal(t, http.StatusUnauthorized, app.do("POST", "/api/v1/auth/login", wrong, "").status)
	assert.Equal(t, http.StatusUnauthorized, app.do("POST", "/api/v1/auth/login", wrong, "").status)

	res := app.do("POST", "/api/v1/auth/login", right, "")
	assert.Equal(t, http.StatusTooManyRequests, res.status, "even the right password is refused while blocked")
	assert.Equal(t, "RATE_LIMITED", res.errorCode())
	assert.NotEmpty(t, res.header.Get("Retry-After"))
}

func TestRegister_RateLimitedPerIP(t *testing.T) {
	app := newTestApp(t, auth.Limits{LoginAttempts: 100, LoginWindow: time.Minute, RegisterAttempts: 2, RegisterWindow: time.Minute})

	app.register("A", "a@example.com", "rahasia-123")
	app.register("B", "b@example.com", "rahasia-123")
	res := app.do("POST", "/api/v1/auth/register", `{"name":"C","email":"c@example.com","password":"rahasia-123"}`, "")

	assert.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
	assert.Equal(t, "RATE_LIMITED", res.errorCode())
	assert.NotEmpty(t, res.header.Get("Retry-After"))
	assert.Equal(t, 2, app.count("SELECT count(*) FROM users"), "the refused sign-up created nothing")
}

// Behind a trusted proxy each real client gets its own sign-up budget; without
// one, a forged X-Forwarded-For does not buy a fresh budget. (httptest
// requests come from 192.0.2.1.)
func TestRegister_RateLimitKeysOnTheRealClientBehindATrustedProxy(t *testing.T) {
	oneSignUp := auth.Limits{LoginAttempts: 100, LoginWindow: time.Minute, RegisterAttempts: 1, RegisterWindow: time.Minute}
	signUp := func(app *testApp, email, xff string) int {
		return app.do("POST", "/api/v1/auth/register",
			`{"name":"X","email":"`+email+`","password":"rahasia-123"}`, "", "X-Forwarded-For", xff).status
	}

	proxied := newTestAppWith(t, oneSignUp, func(c *config.Config) { c.TrustedProxies = []string{"192.0.2.0/24"} })
	assert.Equal(t, http.StatusCreated, signUp(proxied, "a@example.com", "203.0.113.1"))
	assert.Equal(t, http.StatusTooManyRequests, signUp(proxied, "b@example.com", "203.0.113.1"), "same client, budget used")
	assert.Equal(t, http.StatusCreated, signUp(proxied, "c@example.com", "203.0.113.2"), "another client behind the same proxy")

	direct := newTestApp(t, oneSignUp) // TRUSTED_PROXIES empty, the default
	assert.Equal(t, http.StatusCreated, signUp(direct, "d@example.com", "203.0.113.1"))
	assert.Equal(t, http.StatusTooManyRequests, signUp(direct, "e@example.com", "203.0.113.99"), "a forged header is ignored")
}

// APP_ENV=production → every Set-Cookie for the session carries Secure (and
// keeps HttpOnly + SameSite=Lax): register, login and the logout deletion.
// The app is built through cfg.SecureCookies(), the same call main.go makes.
func TestSessionCookie_SecureOnlyInProduction(t *testing.T) {
	prod := newTestAppWith(t, generousLimits, func(c *config.Config) { c.AppEnv = "production" })
	token := prod.register("Rava", "rava@example.com", "rahasia-123")
	login := prod.do("POST", "/api/v1/auth/login", `{"email":"rava@example.com","password":"rahasia-123"}`, token)
	require.Equal(t, http.StatusOK, login.status, login.raw)
	logout := prod.do("POST", "/api/v1/auth/logout", "", login.session().Value)
	require.Equal(t, http.StatusNoContent, logout.status)

	reg := prod.do("POST", "/api/v1/auth/register", `{"name":"B","email":"b@example.com","password":"rahasia-123"}`, "")
	require.Equal(t, http.StatusCreated, reg.status, reg.raw)

	for name, res := range map[string]result{"register": reg, "login": login, "logout": logout} {
		c := res.session()
		require.NotNil(t, c, name)
		assert.True(t, c.Secure, "%s: Secure", name)
		assert.True(t, c.HttpOnly, "%s: HttpOnly", name)
		assert.Equal(t, http.SameSiteLaxMode, c.SameSite, name)
	}

	dev := newTestApp(t, generousLimits)
	res := dev.do("POST", "/api/v1/auth/register", `{"name":"C","email":"c@example.com","password":"rahasia-123"}`, "")
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	assert.False(t, res.session().Secure, "development runs on plain http://localhost")
}

// The background cleanup deletes every expired session, in batches, and
// never an active one.
func TestCleanupExpiredSessions(t *testing.T) {
	app := newTestApp(t, generousLimits)
	ctx := context.Background()
	app.register("A", "a@example.com", "rahasia-123")
	app.register("B", "b@example.com", "rahasia-123")
	active := app.register("C", "c@example.com", "rahasia-123")
	// A second session for A (login without the old cookie keeps the first).
	require.Equal(t, http.StatusOK, app.do("POST", "/api/v1/auth/login", `{"email":"a@example.com","password":"rahasia-123"}`, "").status)

	_, err := app.pool.Exec(ctx, `
		UPDATE user_sessions SET created_at = NOW() - INTERVAL '2 hours', expires_at = NOW() - INTERVAL '1 hour'
		WHERE user_id IN (SELECT id FROM users WHERE email IN ('a@example.com', 'b@example.com'))`)
	require.NoError(t, err)
	require.Equal(t, 4, app.count("SELECT count(*) FROM user_sessions"))

	// Batch of 2 with 3 expired rows: the loop must run twice (2, then 1).
	n, err := app.auth.CleanupExpiredSessions(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
	assert.Equal(t, 1, app.count("SELECT count(*) FROM user_sessions"))
	assert.Equal(t, http.StatusOK, app.do("GET", "/api/v1/me", "", active).status, "the active session survives")

	n, err = app.auth.CleanupExpiredSessions(ctx, 2)
	require.NoError(t, err)
	assert.Zero(t, n, "nothing left to delete")
}

func TestCrossSiteLoginIsRejected(t *testing.T) {
	app := newTestApp(t, generousLimits)
	app.register("Rava", "rava@example.com", "rahasia-123")

	res := app.do("POST", "/api/v1/auth/login", `{"email":"rava@example.com","password":"rahasia-123"}`, "",
		"Origin", "https://evil.example")

	assert.Equal(t, http.StatusForbidden, res.status)
	assert.Equal(t, "ORIGIN_REJECTED", res.errorCode())
	assert.Nil(t, res.session())
}

// ---- profile ----------------------------------------------------------------

func TestUpdateMe(t *testing.T) {
	app := newTestApp(t, generousLimits)
	token := app.register("Rava", "rava@example.com", "rahasia-123")

	assert.Equal(t, http.StatusUnauthorized, app.do("PATCH", "/api/v1/me", `{"name":"X"}`, "").status)

	res := app.do("PATCH", "/api/v1/me", `{"name":"Rava G","phone":"+62 812 3456 7890"}`, token)
	require.Equal(t, http.StatusOK, res.status, res.raw)
	assert.Equal(t, "Rava G", res.data()["name"])
	assert.Equal(t, "+62 812 3456 7890", res.data()["phone"])

	res = app.do("PATCH", "/api/v1/me", `{"phone":""}`, token)
	require.Equal(t, http.StatusOK, res.status)
	assert.Nil(t, res.data()["phone"], "empty string clears the phone")
	assert.Equal(t, "Rava G", res.data()["name"], "omitted fields are unchanged")

	res = app.do("PATCH", "/api/v1/me", `{"phone":"call me"}`, token)
	assert.Equal(t, http.StatusUnprocessableEntity, res.status)

	res = app.do("PATCH", "/api/v1/me", `{"email":"new@example.com"}`, token)
	assert.Equal(t, http.StatusBadRequest, res.status, "email cannot be changed through this endpoint")
}

// ---- database guards (last line of defence, independent of Go code) ---------

func TestDatabaseRejectsInvalidRows(t *testing.T) {
	app := newTestApp(t, generousLimits)
	ctx := context.Background()

	_, err := app.pool.Exec(ctx, "INSERT INTO users (email, password_hash, name) VALUES ('UPPER@example.com', 'x', 'A')")
	assert.ErrorContains(t, err, "users_email_lowercase_check")

	_, err = app.pool.Exec(ctx, "INSERT INTO users (email, password_hash, name, status) VALUES ('a@example.com', 'x', 'A', 'deleted')")
	assert.ErrorContains(t, err, "users_status_check")

	_, err = app.pool.Exec(ctx, "INSERT INTO users (email, password_hash, name) VALUES ('b@example.com', 'x', '   ')")
	assert.ErrorContains(t, err, "users_name_not_blank_check")
}
