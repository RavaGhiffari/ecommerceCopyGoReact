// Package request decodes JSON request bodies strictly.
package request

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
)

// MaxBodyBytes caps request bodies. Our JSON payloads are tiny; anything
// bigger is a mistake or an attempt to exhaust memory.
const MaxBodyBytes = 1 << 20 // 1 MiB

// BindJSON decodes the body into dst. On failure it writes the error response
// itself and returns false, so handlers can simply `if !BindJSON(...) { return }`.
//
// It is strict on purpose:
//   - Content-Type must be application/json. An HTML form on another site can
//     only send form encodings without a CORS preflight, so this also helps
//     against CSRF.
//   - Unknown fields are rejected: a client cannot sneak in {"role":"admin"}
//     and have it silently ignored today and honoured by a future refactor.
//   - Only one JSON value is allowed.
func BindJSON(c *gin.Context, dst any) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		response.Fail(c, http.StatusUnsupportedMediaType, response.CodeUnsupportedMediaType,
			"Content-Type must be application/json", nil)
		return false
	}

	dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			response.Fail(c, http.StatusRequestEntityTooLarge, response.CodePayloadTooLarge,
				"Request body is too large", nil)
			return false
		}
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidJSON, "Request body is not valid JSON for this endpoint", nil)
		return false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidJSON, "Request body must contain a single JSON object", nil)
		return false
	}
	return true
}
