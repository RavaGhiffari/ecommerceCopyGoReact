package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
)

var seller = authctx.Principal{UserID: uuid.New(), Roles: []string{authctx.RoleCustomer, authctx.RoleSeller}}

// fakeResolver knows exactly one valid token.
func fakeResolver(ctx context.Context, raw string) (authctx.Principal, bool, error) {
	switch raw {
	case "valid":
		return seller, true, nil
	case "db-down":
		return authctx.Principal{}, false, errors.New("connection refused")
	default:
		return authctx.Principal{}, false, nil
	}
}

func authRouter() *gin.Engine {
	r := gin.New()
	r.Use(Authenticate(fakeResolver))
	r.GET("/public", func(c *gin.Context) {
		_, ok := authctx.FromContext(c.Request.Context())
		c.String(http.StatusOK, map[bool]string{true: "user", false: "anon"}[ok])
	})
	r.GET("/private", RequireAuth(), func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/seller", RequireRole(authctx.RoleSeller), func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/admin", RequireRole(authctx.RoleAdmin), func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func get(path, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: cookie})
	}
	w := httptest.NewRecorder()
	authRouter().ServeHTTP(w, req)
	return w
}

func TestAuthenticate_AnonymousAndInvalidSessionsContinueWithoutPrincipal(t *testing.T) {
	assert.Equal(t, "anon", get("/public", "").Body.String())
	assert.Equal(t, "anon", get("/public", "expired-or-forged").Body.String())
	assert.Equal(t, "user", get("/public", "valid").Body.String())
}

func TestAuthenticate_DatabaseFailureIs500NotAnonymous(t *testing.T) {
	w := get("/public", "db-down")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "connection refused", "internal errors stay in the logs")
}

func TestRequireAuth(t *testing.T) {
	w := get("/private", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"UNAUTHENTICATED"`))

	assert.Equal(t, http.StatusOK, get("/private", "valid").Code)
}

func TestRequireRole(t *testing.T) {
	assert.Equal(t, http.StatusUnauthorized, get("/seller", "").Code, "anonymous: 401, not 403")
	assert.Equal(t, http.StatusOK, get("/seller", "valid").Code)

	w := get("/admin", "valid")
	assert.Equal(t, http.StatusForbidden, w.Code, "known user without the role: 403")
	assert.Contains(t, w.Body.String(), `"FORBIDDEN"`)
}
