package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func sameOriginRequest(method, host, origin string) int {
	r := gin.New()
	r.Use(SameOrigin([]string{"https://shop.example.com"}))
	r.Any("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(method, "http://"+host+"/x", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestSameOrigin(t *testing.T) {
	tests := []struct {
		name   string
		method string
		host   string
		origin string
		want   int
	}{
		{"cross-site POST is rejected", http.MethodPost, "localhost:8081", "https://evil.example", http.StatusForbidden},
		{"cross-site DELETE is rejected", http.MethodDelete, "localhost:8081", "https://evil.example", http.StatusForbidden},
		{"opaque origin is rejected", http.MethodPost, "localhost:8081", "null", http.StatusForbidden},
		{"same host is allowed (Vite proxy keeps Host)", http.MethodPost, "localhost:5174", "http://localhost:5174", http.StatusOK},
		{"configured origin is allowed", http.MethodPatch, "api.example.com", "https://shop.example.com", http.StatusOK},
		{"no Origin (curl, server-to-server) is allowed", http.MethodPost, "localhost:8081", "", http.StatusOK},
		{"safe methods are never checked", http.MethodGet, "localhost:8081", "https://evil.example", http.StatusOK},
		{"port matters", http.MethodPost, "localhost:5174", "http://localhost:6666", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sameOriginRequest(tt.method, tt.host, tt.origin))
		})
	}
}
