package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"one object", `{"name":"a"}`, false},
		{"trailing whitespace", "{\"name\":\"a\"}\n", false},
		{"two objects", `{"name":"a"}{"name":"b"}`, true},
		{"trailing junk", `{"name":"a"} x`, true},
		{"not json", `x`, true},
		{"empty", ``, true},
		{"too large", `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var dst struct{ Name string }

			err := DecodeJSON(httptest.NewRecorder(), r, &dst)

			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusTeapot, "nope")

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"nope"}` {
		t.Errorf("body = %s", got)
	}
}
