// Command api starts the HTTP server.
//
// main is the "composition root": the only place that knows how every piece
// is built and wired together. Dependencies flow one way:
//
//	config -> database pool -> services -> handlers -> router -> http.Server
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
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
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/logger"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/password"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/router"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(logger.New(os.Stdout, cfg.LogLevel))

	// ctx is cancelled when the process receives Ctrl+C (SIGINT) or
	// `docker compose stop` (SIGTERM). That is our cue to shut down.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close() // runs last: after the server has stopped using it
	slog.Info("connected to database")

	authSvc, err := auth.NewService(pool, auth.Config{
		SessionTTL: cfg.SessionTTL,
		Password: password.Params{
			MemoryKiB:   cfg.Argon2.MemoryKiB,
			Iterations:  cfg.Argon2.Iterations,
			Parallelism: cfg.Argon2.Parallelism,
		},
	})
	if err != nil {
		return err
	}

	checkoutSvc := checkout.NewService(pool, checkout.Config{
		TTL:   cfg.CheckoutTTL,
		Rates: shipping.Rates{Regular: cfg.Shipping.Regular, Express: cfg.Shipping.Express, SameDay: cfg.Shipping.SameDay},
	})

	// The expiry sweeper releases the stock of checkouts nobody paid for. It
	// stops when ctx is cancelled (shutdown signal). The deferred wait runs
	// BEFORE the deferred pool.Close above (defers run last-in, first-out),
	// so the sweeper never uses a closed pool.
	sweeperDone := make(chan struct{})
	go func() {
		defer close(sweeperDone)
		checkoutSvc.RunSweeper(ctx, cfg.CheckoutSweepInterval)
	}()
	defer func() {
		stop() // also covers returning early with an error
		<-sweeperDone
	}()

	handlers := router.Handlers{
		Health:     health.NewHandler(health.NewRepository(pool)),
		Addresses:  addresses.NewHandler(addresses.NewService(pool)),
		Auth:       auth.NewHandler(authSvc, cfg.IsProduction(), auth.DefaultLimits),
		Users:      users.NewHandler(users.NewService(pool)),
		Categories: categories.NewHandler(categories.NewService(pool)),
		Stores:     stores.NewHandler(stores.NewService(pool)),
		Products:   products.NewHandler(products.NewService(pool)),
		Cart:       cart.NewHandler(cart.NewService(pool)),
		Checkout:   checkout.NewHandler(checkoutSvc),
		Orders:     orders.NewHandler(orders.NewService(pool)),
		Reviews:    reviews.NewHandler(reviews.NewService(pool)),
		Wishlist:   wishlist.NewHandler(wishlist.NewService(pool)),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(cfg, handlers, authSvc.ResolveSession),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// Hard backstop: a little longer than the per-request context timeout.
		WriteTimeout: cfg.RequestTimeout + 5*time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Serve in the background so this goroutine can wait for a signal.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("api listening", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err // e.g. port already in use
	case <-ctx.Done():
	}

	// Graceful shutdown: stop accepting new connections, let in-flight
	// requests finish (up to ShutdownTimeout), then return so the deferred
	// pool.Close runs.
	slog.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("shutdown complete")
	return nil
}
