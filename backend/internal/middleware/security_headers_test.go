package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// The router test (TestSecurityHeaders_OnEveryKindOfResponse) checks the
// same values through the full middleware chain.
var wantSecurityHeaders = map[string]string{
	"X-Content-Type-Options":  "nosniff",
	"X-Frame-Options":         "DENY",
	"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	"Referrer-Policy":         "strict-origin-when-cross-origin",
	"Cache-Control":           "no-store",
}

func TestSecurityHeaders_OnSuccessAndOnAbort(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	r.GET("/aborted", func(c *gin.Context) { c.AbortWithStatus(http.StatusForbidden) })

	for _, path := range []string{"/ok", "/aborted"} {
		w := serve(r, httptest.NewRequest(http.MethodGet, path, nil))
		for k, v := range wantSecurityHeaders {
			assert.Equal(t, v, w.Header().Get(k), "%s %s", path, k)
		}
	}
}
