package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
)

// Recovery turns a panic inside a handler into a 500 response.
//
// Without it, one bug (e.g. a nil pointer) would crash the request and Gin
// would answer with an empty body. With it, the client gets our normal error
// envelope, and the stack trace goes to the logs only, never to the client.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// http.ErrAbortHandler is net/http's deliberate "abort this
			// response" signal, not a bug; let net/http handle it.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			slog.ErrorContext(c.Request.Context(), "panic recovered",
				"panic", fmt.Sprint(rec),
				"stack", string(debug.Stack()),
			)

			if c.Writer.Written() {
				// Headers already sent; we can't change the status any more.
				c.Abort()
				return
			}
			response.Fail(c, http.StatusInternalServerError, response.CodeInternal,
				"Internal server error", nil)
		}()

		c.Next()
	}
}
