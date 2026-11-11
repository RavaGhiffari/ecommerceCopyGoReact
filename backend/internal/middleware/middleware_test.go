package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/logger"
)

func init() { gin.SetMode(gin.TestMode) }

func serve(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- RequestID ---------------------------------------------------------------

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	var seen string
	r.GET("/", func(c *gin.Context) { seen = logger.RequestID(c.Request.Context()) })

	w := serve(r, httptest.NewRequest(http.MethodGet, "/", nil))

	id := w.Header().Get(HeaderRequestID)
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{32}$`), id)
	assert.Equal(t, id, seen, "handler context must carry the same ID as the response header")
}

func TestRequestID_KeepsValidClientID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderRequestID, "client-abc_123")
	w := serve(r, req)

	assert.Equal(t, "client-abc_123", w.Header().Get(HeaderRequestID))
}

func TestRequestID_ReplacesUnsafeClientID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {})

	for _, bad := range []string{"has space", `{"fake":"log"}`, string(make([]byte, 65))} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderRequestID, bad)
		w := serve(r, req)

		assert.NotEqual(t, bad, w.Header().Get(HeaderRequestID))
		assert.Len(t, w.Header().Get(HeaderRequestID), 32)
	}
}

// ---- Recovery ----------------------------------------------------------------

func TestRecovery_PanicBecomes500Envelope(t *testing.T) {
	r := gin.New()
	r.Use(Recovery())
	r.GET("/boom", func(c *gin.Context) { panic("secret internal detail") })

	w := serve(r, httptest.NewRequest(http.MethodGet, "/boom", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"success":false,"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`, w.Body.String())
	assert.NotContains(t, w.Body.String(), "secret internal detail", "panic text must never reach the client")
}

// ---- Timeout -----------------------------------------------------------------

func TestTimeout_SetsDeadlineOnRequestContext(t *testing.T) {
	r := gin.New()
	r.Use(Timeout(2 * time.Second))
	var deadline time.Time
	var ok bool
	r.GET("/", func(c *gin.Context) { deadline, ok = c.Request.Context().Deadline() })

	serve(r, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(2*time.Second), deadline, 500*time.Millisecond)
}

// ---- CORS --------------------------------------------------------------------

func corsRouter() *gin.Engine {
	r := gin.New()
	r.Use(CORS([]string{"http://localhost:5173"}))
	r.GET("/data", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func preflight(origin string) *http.Request {
	req := httptest.NewRequest(http.MethodOptions, "/data", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	return req
}

func TestCORS_AllowedOriginPreflight(t *testing.T) {
	w := serve(corsRouter(), preflight("http://localhost:5173"))

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "PATCH")
}

func TestCORS_DisallowedOriginPreflight(t *testing.T) {
	w := serve(corsRouter(), preflight("https://evil.example"))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_SimpleRequests(t *testing.T) {
	r := corsRouter()

	allowed := httptest.NewRequest(http.MethodGet, "/data", nil)
	allowed.Header.Set("Origin", "http://localhost:5173")
	w := serve(r, allowed)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))

	other := httptest.NewRequest(http.MethodGet, "/data", nil)
	other.Header.Set("Origin", "https://evil.example")
	w = serve(r, other)
	assert.Equal(t, http.StatusOK, w.Code, "the server still answers; the browser is what blocks the read")
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))

	noOrigin := httptest.NewRequest(http.MethodGet, "/data", nil)
	w = serve(r, noOrigin)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}
