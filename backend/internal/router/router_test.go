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
	cfg := config.Config{
		AppEnv:             "development",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
		RequestTimeout:     5 * time.Second,
	}
	h := Handlers{Health: health.NewHandler(health.NewRepository(unreachablePool(t)))}
	return New(cfg, h)
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

func TestCORSPreflight_WorksOnRoutesWithoutAnOptionsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/anything", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	w := httptest.NewRecorder()

	testRouter(t).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
}
