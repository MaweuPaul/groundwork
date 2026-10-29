package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func TestLoggerRecordsStatus(t *testing.T) {
	buf := captureLog(t)
	h := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/parcels", nil))

	if !strings.Contains(buf.String(), "POST /parcels 201") {
		t.Errorf("log = %q, want it to contain %q", buf.String(), "POST /parcels 201")
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
