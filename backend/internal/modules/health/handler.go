// Package health exposes liveness and readiness endpoints.
//
// There is no service.go here on purpose: health has no business rules, so a
// service layer would only forward calls. Real domains (auth, products, ...)
// get handler -> service -> repository because they DO have rules to hold.
package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// Live answers "is the process running?". It must not touch the database:
// if PostgreSQL is down, restarting the API would not fix it.
func (h *Handler) Live(c *gin.Context) {
	response.OK(c, http.StatusOK, gin.H{"status": "ok"}, "API is running")
}

// Ready answers "can the API serve real traffic right now?", which requires
// the database. A short timeout keeps the endpoint fast when the DB hangs.
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.repo.Ping(ctx); err != nil {
		// Log the real cause for us; return a generic message to the client.
		slog.ErrorContext(ctx, "readiness check failed", "error", err)
		response.Fail(c, http.StatusServiceUnavailable, response.CodeDatabaseUnavailable,
			"Database is not reachable", nil)
		return
	}

	response.OK(c, http.StatusOK, gin.H{"status": "ready", "database": "up"}, "API is ready")
}
