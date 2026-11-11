package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout puts a deadline on the request's context.
//
// This is cooperative: it does not kill the handler. Anything that respects
// ctx — every pgx query, outgoing HTTP call, or a select on ctx.Done() — stops
// with context.DeadlineExceeded once the deadline passes, so a slow query
// cannot hold a database connection forever. http.Server.WriteTimeout in
// main.go is the hard backstop for code that ignores ctx.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
