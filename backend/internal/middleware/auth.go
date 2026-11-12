package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
)

// SessionCookieName is the cookie holding the raw session token.
const SessionCookieName = "sid"

// SessionResolver turns a raw session token into a principal.
// ok=false means "no valid session" (unknown, expired, suspended user);
// err is reserved for real failures such as the database being down.
// It is a function, not an import of the auth module, so this package does
// not depend on any business module.
type SessionResolver func(ctx context.Context, rawToken string) (p authctx.Principal, ok bool, err error)

// Authenticate identifies the caller from the session cookie, if any.
// It never rejects a request by itself: anonymous requests continue without a
// principal, and RequireAuth / RequireRole decide per route.
func Authenticate(resolve SessionResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(SessionCookieName)
		if err != nil || raw == "" {
			c.Next()
			return
		}

		p, ok, err := resolve(c.Request.Context(), raw)
		if err != nil {
			response.Internal(c, err)
			return
		}
		if ok {
			c.Request = c.Request.WithContext(authctx.WithPrincipal(c.Request.Context(), p))
		}
		c.Next()
	}
}

// RequireAuth rejects anonymous requests with 401.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := authctx.FromContext(c.Request.Context()); !ok {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthenticated, "Authentication required", nil)
			return
		}
		c.Next()
	}
}

// RequireRole allows the request when the caller has at least one of roles.
// 401 = we don't know who you are; 403 = we know, and you may not do this.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := authctx.FromContext(c.Request.Context())
		if !ok {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthenticated, "Authentication required", nil)
			return
		}
		for _, role := range roles {
			if p.HasRole(role) {
				c.Next()
				return
			}
		}
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "You do not have permission to do this", nil)
	}
}
