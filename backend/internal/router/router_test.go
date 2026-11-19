package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/middleware"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/health"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
)

func init() { gin.SetMode(gin.TestMode) }

// unreachablePool returns a pool pointed at a port nothing listens on.
// pgxpool connects lazily, so creating it succeeds; the first query fails.
// That lets us test the "database is down" path without a database.
func unreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://u:p@127.0.0.1:1/db?connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func testRouter(t *testing.T) *gin.Engine {
	return testRouterWith(t, nil)
}

func testRouterWith(t *testing.T, trustedProxies []string) *gin.Engine {
	cfg := config.Config{
		AppEnv:             "development",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
		TrustedProxies:     trustedProxies,
		RequestTimeout:     5 * time.Second,
	}
	// Only the health handler is exercised here; auth flows are covered by the
	// integration tests against a real database.
	h := Handlers{Health: health.NewHandler(health.NewRepository(unreachablePool(t)))}
	noSession := func(context.Context, string) (authctx.Principal, bool, error) {
		return authctx.Principal{}, false, nil
	}
	return New(cfg, h, noSession)
}

func get(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestHealth_IsUpEvenWhenDatabaseIsDown(t *testing.T) {
	w := get(testRouter(t), "/health")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"success":true,"data":{"status":"ok"},"message":"API is running"}`, w.Body.String())
	assert.NotEmpty(t, w.Header().Get(middleware.HeaderRequestID), "middleware chain must be installed")
}

func TestReady_Returns503WhenDatabaseIsDown(t *testing.T) {
	w := get(testRouter(t), "/ready")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.JSONEq(t, `{"success":false,"error":{"code":"DATABASE_UNAVAILABLE","message":"Database is not reachable"}}`, w.Body.String())
}

func TestUnknownRoute_Returns404Envelope(t *testing.T) {
	w := get(testRouter(t), "/api/v1/does-not-exist")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"success":false,"error":{"code":"NOT_FOUND","message":"Route not found"}}`, w.Body.String())
}

func TestWrongMethod_Returns405Envelope(t *testing.T) {
	w := httptest.NewRecorder()
	testRouter(t).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/health", nil))

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.JSONEq(t, `{"success":false,"error":{"code":"METHOD_NOT_ALLOWED","message":"Method not allowed"}}`, w.Body.String())
}

// Every kind of answer gets the security headers: a normal one, a 503, the
// NoRoute/NoMethod envelopes, a refused CORS preflight (aborted inside CORS)
// and a recovered panic.
func TestSecurityHeaders_OnEveryKindOfResponse(t *testing.T) {
	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Cache-Control":           "no-store",
	}
	r := testRouter(t)
	r.GET("/panic", func(*gin.Context) { panic("boom") })

	refused := httptest.NewRequest(http.MethodOptions, "/api/v1/anything", nil)
	refused.Header.Set("Origin", "https://evil.example")
	refused.Header.Set("Access-Control-Request-Method", http.MethodPost)

	cases := map[string]struct {
		req  *http.Request
		code int
	}{
		"health":            {httptest.NewRequest(http.MethodGet, "/health", nil), http.StatusOK},
		"ready (db down)":   {httptest.NewRequest(http.MethodGet, "/ready", nil), http.StatusServiceUnavailable},
		"no route":          {httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil), http.StatusNotFound},
		"no method":         {httptest.NewRequest(http.MethodPost, "/health", nil), http.StatusMethodNotAllowed},
		"refused preflight": {refused, http.StatusForbidden},
		"recovered panic":   {httptest.NewRequest(http.MethodGet, "/panic", nil), http.StatusInternalServerError},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, tc.req)
			require.Equal(t, tc.code, w.Code)
			for k, v := range want {
				assert.Equal(t, v, w.Header().Get(k), k)
			}
		})
	}
}

// ClientIP() is the rate-limit key. httptest requests come from 192.0.2.1.
func TestClientIP_TrustedProxies(t *testing.T) {
	ipOf := func(r *gin.Engine, xff string) string {
		var ip string
		r.GET("/_ip", func(c *gin.Context) { ip = c.ClientIP() })
		req := httptest.NewRequest(http.MethodGet, "/_ip", nil)
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		r.ServeHTTP(httptest.NewRecorder(), req)
		return ip
	}

	// Default (empty TRUSTED_PROXIES): the header is the client's own words
	// and is ignored, so a client cannot pick a fresh rate-limit bucket.
	assert.Equal(t, "192.0.2.1", ipOf(testRouterWith(t, nil), "203.0.113.7"))

	// The peer is a trusted proxy: the address it appended is the client.
	assert.Equal(t, "203.0.113.7", ipOf(testRouterWith(t, []string{"192.0.2.0/24"}), "203.0.113.7"))

	// A client that forges the header itself still ends up as the address
	// the trusted proxy saw (the right-most untrusted entry), not its forgery.
	assert.Equal(t, "203.0.113.7", ipOf(testRouterWith(t, []string{"192.0.2.0/24"}), "10.9.9.9, 203.0.113.7"))

	// Trusting some other address does not make this peer trusted.
	assert.Equal(t, "192.0.2.1", ipOf(testRouterWith(t, []string{"10.0.0.1"}), "203.0.113.7"))
}

func TestCORSPreflight_WorksOnRoutesWithoutAnOptionsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/anything", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	w := httptest.NewRecorder()

	testRouter(t).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
}
