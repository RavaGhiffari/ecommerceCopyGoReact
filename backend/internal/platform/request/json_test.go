package request

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

type body struct {
	Name string `json:"name"`
}

func bind(contentType, payload string) (*httptest.ResponseRecorder, body, bool) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}
	var b body
	ok := BindJSON(c, &b)
	return w, b, ok
}

func TestBindJSON_Valid(t *testing.T) {
	_, b, ok := bind("application/json; charset=utf-8", `{"name":"Rava"}`)
	assert.True(t, ok)
	assert.Equal(t, "Rava", b.Name)
}

func TestBindJSON_Rejections(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		payload     string
		status      int
		code        string
	}{
		{"form encoding (what a cross-site form sends)", "application/x-www-form-urlencoded", "name=x", 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"missing content type", "", `{"name":"x"}`, 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"broken JSON", "application/json", `{"name":`, 400, "INVALID_JSON"},
		{"unknown field such as role", "application/json", `{"name":"x","role":"admin"}`, 400, "INVALID_JSON"},
		{"two JSON values", "application/json", `{"name":"x"}{"name":"y"}`, 400, "INVALID_JSON"},
		{"too large", "application/json", `{"name":"` + strings.Repeat("a", MaxBodyBytes) + `"}`, 413, "PAYLOAD_TOO_LARGE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, ok := bind(tt.contentType, tt.payload)
			assert.False(t, ok)
			assert.Equal(t, tt.status, w.Code)
			assert.Contains(t, w.Body.String(), `"`+tt.code+`"`)
		})
	}
}
