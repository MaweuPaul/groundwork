package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"groundwork/internal/httpx"
	"groundwork/internal/logging"
)

// statusRecorder remembers the status code a handler wrote, for the log line.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// RequestID tags the request's logger with a new ID and returns it in X-Request-ID, so a
// failure someone reports can be found in the logs. An ID sent by the client is ignored.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(logging.With(r.Context(), "request_id", id)))
	})
}

// Logger writes one line per request. The path is logged without its query string.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		level := slog.LevelInfo
		if rec.status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		logging.From(r.Context()).Log(r.Context(), level, "request",
			"method", r.Method, "path", r.URL.Path, "status", rec.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

// Recover turns a panic in any handler into a logged 500 instead of a dropped connection.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logging.From(r.Context()).Error("panic",
					"method", r.Method, "path", r.URL.Path, "error", err, "stack", string(debug.Stack()))
				httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
