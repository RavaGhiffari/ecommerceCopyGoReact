// Package middleware holds cross-cutting HTTP concerns that run around every
// handler. Order matters and is set in internal/router: request ID first (so
// everything after can log it), then logging, then recovery, then the rest.
package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/logger"
)

const HeaderRequestID = "X-Request-ID"

// RequestID gives every request an ID, echoes it in the response header and
// stores it in the request context for logging.
//
// A client (or a future reverse proxy) may send its own X-Request-ID so one ID
// can follow a request across systems. We only accept short IDs of safe
// characters: the value ends up in our logs, and accepting arbitrary text
// would let a client inject fake log content.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if !isValidRequestID(id) {
			id = newRequestID()
		}

		c.Header(HeaderRequestID, id)
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

func isValidRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return false
		}
	}
	return true
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error on supported platforms
	return hex.EncodeToString(b)
}
