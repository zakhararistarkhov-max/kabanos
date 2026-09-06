// Package observability wires structured logging and health/readiness probes.
// It intentionally leans on the standard library (log/slog) so there is no
// vendor lock-in; the JSON output is directly consumable by Loki, Datadog, etc.
package observability

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey int

const loggerKey ctxKey = iota

// NewLogger returns a slog.Logger configured for the environment, tagged with
// the given service name. Development uses human-readable text; production uses
// JSON for log aggregation.
func NewLogger(env, service string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler).With(slog.String("service", service))
}

// WithLogger stores a request-scoped logger (carrying request id, user id, …)
// on the context so handlers can log with full correlation.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// LoggerFrom retrieves the request-scoped logger, falling back to the default.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
