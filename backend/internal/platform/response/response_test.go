package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

func TestOK_WritesSuccessEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, http.StatusCreated, map[string]string{"id": "abc"}, "Created")

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.JSONEq(t, `{"success":true,"data":{"id":"abc"},"message":"Created"}`, w.Body.String())
}

func TestFail_WritesErrorEnvelopeAndAborts(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid request",
		map[string]string{"email": "is required"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.True(t, c.IsAborted(), "Fail must stop later handlers from running")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["success"])
	assert.NotContains(t, body, "data")
	assert.Equal(t, map[string]any{
		"code":    "VALIDATION_ERROR",
		"message": "Invalid request",
		"details": map[string]any{"email": "is required"},
	}, body["error"])
}

func TestFail_OmitsEmptyDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, http.StatusNotFound, CodeNotFound, "Route not found", nil)

	assert.JSONEq(t, `{"success":false,"error":{"code":"NOT_FOUND","message":"Route not found"}}`, w.Body.String())
}
