package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_AddsRequestIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo).With("component", "test") // With must keep the behaviour

	ctx := WithRequestID(context.Background(), "req-123")
	log.InfoContext(ctx, "hello")

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))
	assert.Equal(t, "req-123", line["request_id"])
	assert.Equal(t, "test", line["component"])
	assert.Equal(t, "hello", line["msg"])
}

func TestLogger_NoRequestIDWithoutContext(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelInfo).Info("hello")

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))
	assert.NotContains(t, line, "request_id")
}

func TestLogger_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelWarn).Info("too quiet to log")

	assert.Empty(t, buf.String())
}
