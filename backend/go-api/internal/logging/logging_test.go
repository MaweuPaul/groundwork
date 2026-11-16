package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestSetupRejectsBadSettings(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	tests := []struct{ name, level, format string }{
		{"bad level", "loud", "text"},
		{"bad format", "info", "xml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Setup(tt.level, tt.format); err == nil {
				t.Errorf("Setup(%q, %q) = nil, want an error", tt.level, tt.format)
			}
		})
	}
	if err := Setup("debug", "json"); err != nil {
		t.Errorf("Setup(debug, json) = %v", err)
	}
}

func TestFromFallsBackToDefault(t *testing.T) {
	if From(context.Background()) != slog.Default() {
		t.Error("From without a request logger should return the default")
	}
}

func TestWithAddsFieldsToLaterLines(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	ctx := With(With(context.Background(), "request_id", "abc"), "user_id", "u1")
	From(ctx).Info("hello")

	for _, want := range []string{"request_id=abc", "user_id=u1", "msg=hello"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log = %q, want it to contain %q", buf.String(), want)
		}
	}
}
