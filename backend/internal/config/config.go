// Package config loads application settings from environment variables.
//
// Every setting comes from the environment so the same binary runs locally,
// in Docker Compose, and later in production without code changes. Load fails
// fast on missing or malformed values: it is better for the API to refuse to
// start than to start half-configured.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Config struct {
	AppEnv   string     // "development" or "production"
	Port     string     // HTTP port the API listens on, e.g. "8081"
	LogLevel slog.Level // debug, info, warn, error

	// Browser origins allowed to call the API cross-origin, e.g. http://localhost:5173.
	// In development the Vite proxy makes requests same-origin, so this is a backstop.
	CORSAllowedOrigins []string

	RequestTimeout  time.Duration // max time a handler's context stays valid
	ShutdownTimeout time.Duration // how long to wait for in-flight requests on shutdown

	SessionTTL time.Duration // how long a login stays valid
	Argon2     Argon2Config
	RateLimits RateLimits

	CheckoutTTL time.Duration // how long an unpaid checkout holds its stock
	// How often the API looks for checkouts whose time ran out.
	CheckoutSweepInterval time.Duration
	Shipping              ShippingFees

	DB DBConfig
}

// Argon2Config is the cost of hashing one password (see platform/password).
type Argon2Config struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
}

// RateLimits are the brute-force limits of login and register (see
// modules/auth). The defaults are the production values; only the E2E stack
// raises them, because every browser test signs up from the same IP.
type RateLimits struct {
	LoginFailures  int // failed logins allowed per IP + email within LoginWindow
	LoginWindow    time.Duration
	Registrations  int // sign-ups allowed per IP within RegisterWindow
	RegisterWindow time.Duration
}

// ShippingFees are the flat fees per store order (simulated courier, Phase 5).
// Money is parsed as an exact decimal, never a float.
type ShippingFees struct {
	Regular decimal.Decimal
	Express decimal.Decimal
	SameDay decimal.Decimal
}

// DBConfig controls the pgx connection pool. See internal/database.
type DBConfig struct {
	URL               string        // postgres://user:pass@host:port/db?sslmode=disable
	MaxConns          int32         // upper bound of open connections
	MinConns          int32         // connections kept open even when idle
	MaxConnLifetime   time.Duration // recycle a connection after this long
	MaxConnIdleTime   time.Duration // close idle connections after this long
	HealthCheckPeriod time.Duration // how often the pool checks idle connections
	ConnectTimeout    time.Duration // timeout for establishing one connection
}

func Load() (Config, error) {
	var errs []error

	cfg := Config{
		AppEnv:             getString("APP_ENV", "development"),
		Port:               getString("APP_PORT", "8081"),
		LogLevel:           getLogLevel("LOG_LEVEL", slog.LevelInfo, &errs),
		CORSAllowedOrigins: getOrigins("CORS_ALLOWED_ORIGINS", "http://localhost:5173", &errs),
		RequestTimeout:     getDuration("REQUEST_TIMEOUT", 10*time.Second, &errs),
		ShutdownTimeout:    getDuration("SHUTDOWN_TIMEOUT", 15*time.Second, &errs),
		SessionTTL:         getDuration("SESSION_TTL", 24*time.Hour, &errs),
		Argon2: Argon2Config{
			// Defaults: OWASP minimum for Argon2id (19 MiB, t=2, p=1).
			MemoryKiB:   uint32(getInt32("ARGON2_MEMORY_KIB", 19*1024, &errs)),
			Iterations:  uint32(getInt32("ARGON2_ITERATIONS", 2, &errs)),
			Parallelism: uint8(min(getInt32("ARGON2_PARALLELISM", 1, &errs), 255)),
		},
		RateLimits: RateLimits{
			LoginFailures:  int(getInt32("RATE_LIMIT_LOGIN_FAILURES", 5, &errs)),
			LoginWindow:    getDuration("RATE_LIMIT_LOGIN_WINDOW", 15*time.Minute, &errs),
			Registrations:  int(getInt32("RATE_LIMIT_REGISTRATIONS", 10, &errs)),
			RegisterWindow: getDuration("RATE_LIMIT_REGISTER_WINDOW", time.Hour, &errs),
		},
		CheckoutTTL:           getDuration("CHECKOUT_TTL", 30*time.Minute, &errs),
		CheckoutSweepInterval: getDuration("CHECKOUT_SWEEP_INTERVAL", time.Minute, &errs),
		Shipping: ShippingFees{
			Regular: getMoney("SHIPPING_FEE_REGULAR", 15_000, &errs),
			Express: getMoney("SHIPPING_FEE_EXPRESS", 30_000, &errs),
			SameDay: getMoney("SHIPPING_FEE_SAME_DAY", 45_000, &errs),
		},
		DB: DBConfig{
			URL:               os.Getenv("DATABASE_URL"),
			MaxConns:          getInt32("DB_MAX_CONNS", 10, &errs),
			MinConns:          getInt32("DB_MIN_CONNS", 2, &errs),
			MaxConnLifetime:   getDuration("DB_MAX_CONN_LIFETIME", time.Hour, &errs),
			MaxConnIdleTime:   getDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute, &errs),
			HealthCheckPeriod: getDuration("DB_HEALTH_CHECK_PERIOD", time.Minute, &errs),
			ConnectTimeout:    getDuration("DB_CONNECT_TIMEOUT", 5*time.Second, &errs),
		},
	}

	if cfg.DB.URL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if cfg.AppEnv != "development" && cfg.AppEnv != "production" {
		errs = append(errs, fmt.Errorf("APP_ENV must be development or production, got %q", cfg.AppEnv))
	}
	if cfg.Argon2.MemoryKiB < 8*1024 || cfg.Argon2.Iterations < 1 || cfg.Argon2.Parallelism < 1 {
		errs = append(errs, errors.New("ARGON2_* settings are below the safe minimum (8 MiB memory, 1 iteration, 1 thread)"))
	}
	// 0 would lock everyone out (or, read the other way, mean "no limit");
	// neither is something to get by accident from a typo.
	if cfg.RateLimits.LoginFailures < 1 || cfg.RateLimits.Registrations < 1 {
		errs = append(errs, errors.New("RATE_LIMIT_LOGIN_FAILURES and RATE_LIMIT_REGISTRATIONS must be at least 1"))
	}
	if cfg.DB.MinConns > cfg.DB.MaxConns {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS (%d) cannot exceed DB_MAX_CONNS (%d)", cfg.DB.MinConns, cfg.DB.MaxConns))
	}

	return cfg, errors.Join(errs...)
}

func (c Config) IsProduction() bool { return c.AppEnv == "production" }

func getString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getLogLevel(key string, fallback slog.Level, errs *[]error) slog.Level {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(v)); err != nil {
		*errs = append(*errs, fmt.Errorf("%s must be debug, info, warn or error, got %q", key, v))
		return fallback
	}
	return level
}

// getOrigins parses a comma-separated list of exact origins.
// "*" is rejected: the API will send cookies (credentials), and browsers
// forbid a wildcard origin together with credentials for good reason.
func getOrigins(key, fallback string, errs *[]error) []string {
	raw := getString(key, fallback)
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "" {
			continue
		}
		if o == "*" || !(strings.HasPrefix(o, "http://") || strings.HasPrefix(o, "https://")) {
			*errs = append(*errs, fmt.Errorf("%s entries must be exact http(s) origins, got %q", key, o))
			continue
		}
		origins = append(origins, o)
	}
	return origins
}

func getInt32(key string, fallback int32, errs *[]error) int32 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil || n < 0 {
		*errs = append(*errs, fmt.Errorf("%s must be a non-negative integer, got %q", key, v))
		return fallback
	}
	return int32(n)
}

func getDuration(key string, fallback time.Duration, errs *[]error) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		*errs = append(*errs, fmt.Errorf("%s must be a positive duration like 30s or 5m, got %q", key, v))
		return fallback
	}
	return d
}

// getMoney parses a non-negative amount with at most 2 decimals, e.g. "15000".
func getMoney(key string, fallback int64, errs *[]error) decimal.Decimal {
	v := os.Getenv(key)
	if v == "" {
		return decimal.NewFromInt(fallback)
	}
	d, err := decimal.NewFromString(v)
	if err != nil || d.IsNegative() || d.Exponent() < -2 {
		*errs = append(*errs, fmt.Errorf("%s must be a non-negative amount with at most 2 decimals, got %q", key, v))
		return decimal.NewFromInt(fallback)
	}
	return d
}
