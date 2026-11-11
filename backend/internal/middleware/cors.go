package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS lets browsers on an allowed origin (e.g. a separately hosted frontend)
// call this API and send cookies with the request.
//
// What CORS is and isn't: it only tells BROWSERS whether a page on another
// origin may read our responses. It does not stop curl, scripts or attackers
// calling the API directly, so it is never a substitute for authentication,
// authorization or CSRF protection (Phase 1).
//
// Written by hand (~40 lines) instead of adding gin-contrib/cors so the rules
// are visible: exact origin match only, no wildcard, credentials allowed.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			// Same-origin request or non-browser client: CORS does not apply.
			c.Next()
			return
		}

		// Responses differ per Origin, so caches must not mix them up.
		c.Header("Vary", "Origin")

		_, ok := allowed[origin]
		isPreflight := c.Request.Method == http.MethodOptions &&
			c.GetHeader("Access-Control-Request-Method") != ""

		if !ok {
			if isPreflight {
				// No CORS headers: the browser will block the real request.
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", HeaderRequestID)

		if isPreflight {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, X-CSRF-Token")
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
