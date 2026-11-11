package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging writes one structured log line per request after it finishes.
//
// It logs the path but NOT the query string or body: those can carry
// passwords, tokens or personal data, and logs are kept longer and seen by
// more people than the database.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next() // run the rest of the chain (recovery, handler, ...)

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		slog.LogAttrs(c.Request.Context(), level, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("route", c.FullPath()), // "/products/:id" form; empty for unknown routes
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.Int("bytes", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
