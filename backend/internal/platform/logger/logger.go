// Package logger sets up structured JSON logging and request correlation.
//
// Every log line is one JSON object, so tools can filter by field
// (level, request_id, status, ...). The request ID middleware stores an ID in
// the request's context.Context; this package's handler reads it back and adds
// "request_id" to every log written with slog.*Context(ctx, ...). That lets you
// grep one request's whole story out of the logs.
package logger

import (
	"context"
	"io"
	"log/slog"
)

type requestIDKey struct{}

// WithRequestID returns a copy of ctx carrying the request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the request ID stored in ctx, or "" if none.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// New returns a JSON logger that adds request_id from the context.
func New(w io.Writer, level slog.Level) *slog.Logger {
	base := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(contextHandler{base})
}

// contextHandler wraps another slog.Handler and enriches each record.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs and WithGroup must re-wrap, otherwise derived loggers
// (logger.With(...)) would silently lose the request_id behaviour.
func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
