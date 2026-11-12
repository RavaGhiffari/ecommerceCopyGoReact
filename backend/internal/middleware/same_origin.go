package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
)

// SameOrigin is our CSRF defence for state-changing requests.
//
// The attack: you are logged in here, then visit evil.example, which submits
// a hidden request to our API. The browser attaches your session cookie
// automatically. Three layers stop it:
//
//  1. The session cookie is SameSite=Lax: browsers do not send it on
//     cross-site POST/PUT/PATCH/DELETE at all.
//  2. This middleware: browsers always send an Origin header on those methods.
//     If it names another site, we refuse with 403 before any handler runs.
//  3. request.BindJSON requires Content-Type: application/json, which a plain
//     cross-site form cannot send without a CORS preflight we would refuse.
//
// Accepted origins: the same host the request was sent to (normal case, also
// through the Vite proxy) or one listed in CORS_ALLOWED_ORIGINS.
// No Origin header means a non-browser client (curl, server-to-server); those
// carry no ambient browser cookies, so CSRF does not apply to them.
func SameOrigin(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next() // safe methods must not change state, so there is nothing to forge
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := allowed[origin]; ok {
			c.Next()
			return
		}
		if u, err := url.Parse(origin); err == nil && u.Host != "" && u.Host == c.Request.Host {
			c.Next()
			return
		}

		response.Fail(c, http.StatusForbidden, response.CodeOriginRejected, "Cross-site request rejected", nil)
	}
}
