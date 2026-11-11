// Package router builds the Gin engine and registers every route.
// Reading this one file shows the whole public surface of the API.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/middleware"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/health"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
)

type Handlers struct {
	Health *health.Handler
}

func New(cfg config.Config, h Handlers) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// We are not behind a reverse proxy yet, so never trust X-Forwarded-For.
	_ = r.SetTrustedProxies(nil)

	// Middleware runs top to bottom on the way in, bottom to top on the way out:
	//   RequestID -> Logging -> Recovery -> CORS -> Timeout -> handler
	// Logging sits outside Recovery so a recovered panic is still logged as a 500.
	r.Use(
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORSAllowedOrigins),
		middleware.Timeout(cfg.RequestTimeout),
	)

	// Unknown routes still answer in the standard envelope.
	r.HandleMethodNotAllowed = true
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "Route not found", nil)
	})
	r.NoMethod(func(c *gin.Context) {
		response.Fail(c, http.StatusMethodNotAllowed, response.CodeMethodNotAllowed, "Method not allowed", nil)
	})

	// Health endpoints sit outside /api/v1: they are for infrastructure, not clients.
	r.GET("/health", h.Health.Live)
	r.GET("/ready", h.Health.Ready)

	// Business endpoints are registered under /api/v1 starting in Phase 1.
	_ = r.Group("/api/v1")

	return r
}
