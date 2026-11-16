// Package logging is the API's one logger. Setup installs it as the slog default, and each
// request carries a copy tagged with its request ID, so use From(ctx) wherever there is a context.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

// Setup also routes the standard log package through slog, so nothing prints in another format.
func Setup(level, format string) error {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, not %q", level)
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		return fmt.Errorf("LOG_FORMAT must be text or json, not %q", format)
	}

	slog.SetDefault(slog.New(handler))
	return nil
}

// From returns the request's logger, or the default one outside a request.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// With adds fields to every line logged through the returned context.
func With(ctx context.Context, args ...any) context.Context {
	return context.WithValue(ctx, ctxKey{}, From(ctx).With(args...))
}
