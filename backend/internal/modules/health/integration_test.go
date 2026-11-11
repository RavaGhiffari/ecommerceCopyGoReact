//go:build integration

// Integration tests talk to a real PostgreSQL. They only compile with
// `-tags=integration` (see `make test-integration`), so `make test` stays
// fast and works without a database.
package health_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/health"
)

func testDBConfig(t *testing.T) config.DBConfig {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		// Fail, don't skip: this file only runs when integration tests were
		// explicitly requested, so a missing DB is a setup error to fix.
		t.Fatal("TEST_DATABASE_URL is not set; run `make test-integration`")
	}
	return config.DBConfig{
		URL: url, MaxConns: 2, MinConns: 0,
		MaxConnLifetime: time.Minute, MaxConnIdleTime: time.Minute,
		HealthCheckPeriod: time.Minute, ConnectTimeout: 5 * time.Second,
	}
}

func TestRepositoryPing_AgainstRealDatabase(t *testing.T) {
	pool, err := database.NewPool(context.Background(), testDBConfig(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	repo := health.NewRepository(pool)
	assert.NoError(t, repo.Ping(context.Background()))
}

func TestReadyEndpoint_AgainstRealDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool, err := database.NewPool(context.Background(), testDBConfig(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	r := gin.New()
	r.GET("/ready", health.NewHandler(health.NewRepository(pool)).Ready)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"success":true,"data":{"status":"ready","database":"up"},"message":"API is ready"}`, w.Body.String())
}

func TestNewPool_FailsFastWithWrongCredentials(t *testing.T) {
	cfg := testDBConfig(t)
	cfg.URL = "postgres://wrong_user:wrong_pass@postgres:5432/postgres?sslmode=disable"

	_, err := database.NewPool(context.Background(), cfg)
	assert.Error(t, err, "startup must fail instead of serving with a broken database")
}
