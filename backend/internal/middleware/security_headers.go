package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders adds response headers that tell browsers to be stricter
// with what the API sends. The API only ever returns JSON, so they cost
// nothing and close a few browser-side holes:
//
//   - X-Content-Type-Options: nosniff — the browser must trust our
//     Content-Type and never "sniff" a JSON body (which can echo user text,
//     e.g. a review comment) as HTML or script.
//   - X-Frame-Options: DENY and CSP frame-ancestors 'none' — no site may put
//     an API response in a frame (clickjacking). The CSP form is the modern
//     one; X-Frame-Options covers old browsers.
//   - Content-Security-Policy default-src 'none' — if a response is ever
//     opened as a document, it may load nothing at all. The SPA's own CSP
//     (scripts, styles, images) belongs to whatever serves index.html: nginx
//     in production (Phase 7e), Vite in development.
//   - Referrer-Policy: strict-origin-when-cross-origin — another site never
//     sees our full URLs (paths can carry ids and search terms).
//   - Cache-Control: no-store — responses carry personal data (profile,
//     orders, addresses); no browser or shared cache may keep them.
//
// HSTS is not set here: it only means something over HTTPS, and TLS ends in
// front of the API (the proxy sets it).
//
// The headers are set BEFORE c.Next(), so every response gets them —
// including 404, 405, a recovered panic (500) and a CORS-refused preflight.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}
