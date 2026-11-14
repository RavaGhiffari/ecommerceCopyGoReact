package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnv makes each test independent of the machine's real environment.
// t.Setenv restores the original values when the test ends.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"APP_ENV", "APP_PORT", "LOG_LEVEL", "CORS_ALLOWED_ORIGINS",
		"REQUEST_TIMEOUT", "SHUTDOWN_TIMEOUT", "DATABASE_URL",
		"DB_MAX_CONNS", "DB_MIN_CONNS", "DB_MAX_CONN_LIFETIME",
		"DB_MAX_CONN_IDLE_TIME", "DB_HEALTH_CHECK_PERIOD", "DB_CONNECT_TIMEOUT",
		"SESSION_TTL", "ARGON2_MEMORY_KIB", "ARGON2_ITERATIONS", "ARGON2_PARALLELISM",
		"CHECKOUT_TTL", "CHECKOUT_SWEEP_INTERVAL", "SHIPPING_FEE_REGULAR", "SHIPPING_FEE_EXPRESS", "SHIPPING_FEE_SAME_DAY",
	} {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.Port)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
	assert.Equal(t, []string{"http://localhost:5173"}, cfg.CORSAllowedOrigins)
	assert.Equal(t, 10*time.Second, cfg.RequestTimeout)
	assert.Equal(t, int32(10), cfg.DB.MaxConns)
	assert.Equal(t, int32(2), cfg.DB.MinConns)
	assert.False(t, cfg.IsProduction())
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	clearEnv(t)

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is required")
}

func TestLoad_ReportsEveryInvalidValue(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("APP_ENV", "staging")
	t.Setenv("DB_MAX_CONNS", "ten")
	t.Setenv("REQUEST_TIMEOUT", "10") // missing unit
	t.Setenv("LOG_LEVEL", "loud")

	_, err := Load()
	require.Error(t, err)

	msg := err.Error()
	for _, want := range []string{"APP_ENV", "DB_MAX_CONNS", "REQUEST_TIMEOUT", "LOG_LEVEL"} {
		assert.Contains(t, msg, want, "all problems should be reported at once, not one per restart")
	}
}

func TestLoad_MinConnsCannotExceedMax(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("DB_MAX_CONNS", "2")
	t.Setenv("DB_MIN_CONNS", "5")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_MIN_CONNS")
}

func TestLoad_CORSOrigins(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    []string
		wantErr bool
	}{
		{name: "list with spaces and trailing slash", value: " http://localhost:5173/ , https://shop.example.com",
			want: []string{"http://localhost:5173", "https://shop.example.com"}},
		{name: "wildcard rejected", value: "*", wantErr: true},
		{name: "missing scheme rejected", value: "localhost:5173", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
			t.Setenv("CORS_ALLOWED_ORIGINS", tt.value)

			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS"))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.CORSAllowedOrigins)
		})
	}
}

func TestLoad_CheckoutAndShipping(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 30*time.Minute, cfg.CheckoutTTL)
	assert.Equal(t, time.Minute, cfg.CheckoutSweepInterval)
	assert.Equal(t, "15000", cfg.Shipping.Regular.String())

	t.Setenv("CHECKOUT_SWEEP_INTERVAL", "0s")
	_, err = Load()
	assert.ErrorContains(t, err, "CHECKOUT_SWEEP_INTERVAL", "a zero interval would make the ticker panic")
	t.Setenv("CHECKOUT_SWEEP_INTERVAL", "")
	assert.Equal(t, "45000", cfg.Shipping.SameDay.String())

	t.Setenv("SHIPPING_FEE_EXPRESS", "25000.50")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, "25000.5", cfg.Shipping.Express.String())

	for _, bad := range []string{"-1", "abc", "1.005"} {
		t.Setenv("SHIPPING_FEE_EXPRESS", bad)
		_, err = Load()
		assert.ErrorContains(t, err, "SHIPPING_FEE_EXPRESS", bad)
	}
}
