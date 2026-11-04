package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"groundwork/internal/logging"
)

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestLoggerRecordsStatus(t *testing.T) {
	buf := captureLog(t)
	h := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/parcels", nil))

	for _, want := range []string{"level=INFO", "method=POST", "path=/parcels", "status=201"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log = %q, want it to contain %q", buf.String(), want)
		}
	}
}

func TestLoggerMarksServerErrors(t *testing.T) {
	buf := captureLog(t)
	h := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/parcels", nil))

	if !strings.Contains(buf.String(), "level=ERROR") {
		t.Errorf("log = %q, want level=ERROR", buf.String())
	}
}

func TestRequestIDReachesResponseAndLogs(t *testing.T) {
	buf := captureLog(t)
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logging.From(r.Context()).Info("inside")
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/parcels", nil)
	req.Header.Set("X-Request-ID", "client-sent")

	h.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" || id == "client-sent" {
		t.Fatalf("X-Request-ID = %q, want a new server ID", id)
	}
	if !strings.Contains(buf.String(), "request_id="+id) {
		t.Errorf("log = %q, want it to contain the request ID %s", buf.String(), id)
	}
}

func TestRecoverTurnsPanicInto500(t *testing.T) {
	buf := captureLog(t)
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/parcels", nil))

	assertError(t, rec, http.StatusInternalServerError, "something went wrong")
	if !strings.Contains(buf.String(), "boom") {
		t.Errorf("panic was not logged: %q", buf.String())
	}
}
