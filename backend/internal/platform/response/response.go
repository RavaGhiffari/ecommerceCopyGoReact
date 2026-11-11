// Package response writes every API reply in one consistent JSON envelope:
//
//	success: { "success": true,  "data": {...}, "message": "..." }
//	error:   { "success": false, "error": { "code": "...", "message": "...", "details": {...} } }
//
// Handlers call OK / Fail instead of c.JSON directly so the shape never drifts
// between endpoints, and the frontend can rely on it.
package response

import "github.com/gin-gonic/gin"

type successBody struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

type errorBody struct {
	Success bool      `json:"success"`
	Error   ErrorInfo `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`              // machine-readable, e.g. "DATABASE_UNAVAILABLE"
	Message string `json:"message"`           // safe for end users; never raw SQL or stack traces
	Details any    `json:"details,omitempty"` // optional, e.g. per-field validation errors
}

// Error codes shared across modules. Modules may define their own as well.
const (
	CodeInternal            = "INTERNAL_ERROR"
	CodeNotFound            = "NOT_FOUND"
	CodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	CodeDatabaseUnavailable = "DATABASE_UNAVAILABLE"
)

func OK(c *gin.Context, status int, data any, message string) {
	c.JSON(status, successBody{Success: true, Data: data, Message: message})
}

func Fail(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, errorBody{
		Success: false,
		Error:   ErrorInfo{Code: code, Message: message, Details: details},
	})
}
