package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"groundwork/internal/ratelimit"
)

const secret = "0123456789abcdef0123456789abcdef"

func TestClientIP(t *testing.T) {
	tests := []struct {
		name        string
		secret      string
		behindProxy bool
		headers     map[string]string
		want        string
	}{
		{"connection address", "", false, nil, "10.0.0.9"},
		{"forwarded header ignored without a proxy", "", false,
			map[string]string{"X-Forwarded-For": "1.1.1.1"}, "10.0.0.9"},
		{"proxy's own entry, not the caller's", "", true,
			map[string]string{"X-Forwarded-For": "6.6.6.6, 203.0.113.7"}, "203.0.113.7"},
		{"frontend's address with the secret", secret, true,
			map[string]string{"X-Proxy-Secret": secret, "X-Client-IP": "198.51.100.4", "X-Forwarded-For": "203.0.113.7"}, "198.51.100.4"},
		{"frontend's address with a wrong secret", secret, true,
			map[string]string{"X-Proxy-Secret": "guess", "X-Client-IP": "198.51.100.4", "X-Forwarded-For": "203.0.113.7"}, "203.0.113.7"},
		{"client address sent when no secret is configured", "", true,
			map[string]string{"X-Proxy-Secret": "", "X-Client-IP": "198.51.100.4", "X-Forwarded-For": "203.0.113.7"}, "203.0.113.7"},
		{"unparseable client address", secret, false,
			map[string]string{"X-Proxy-Secret": secret, "X-Client-IP": "not-an-ip"}, "10.0.0.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
			r.RemoteAddr = "10.0.0.9:51234"
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			if got := ClientIP(tt.secret, tt.behindProxy)(r); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	limiter := ratelimit.New(2, time.Hour)
	key := func(r *http.Request) string { return r.Header.Get("K") }
	h := RateLimit(limiter, key, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	send := func(k string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
		r.Header.Set("K", k)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	send("a")
	send("a")
	w := send("a")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request: status %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") != "3600" {
		t.Fatalf("Retry-After %q, want 3600", w.Header().Get("Retry-After"))
	}
	if !strings.Contains(w.Body.String(), "Try again in 1 hour.") {
		t.Fatalf("body %s", w.Body.String())
	}
	if w := send(""); w.Code != http.StatusNoContent {
		t.Fatalf("empty key: status %d, want it passed through", w.Code)
	}
}

func TestEmailKeyKeepsBody(t *testing.T) {
	body := `{"email":"  Ana@Example.COM ","password":"x"}`
	r := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))

	if got := EmailKey(r); got != "ana@example.com" {
		t.Fatalf("key %q", got)
	}
	rest, _ := io.ReadAll(r.Body)
	if string(rest) != body {
		t.Fatalf("handler would read %q, want the original body", rest)
	}

	if got := EmailKey(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))); got != "" {
		t.Fatalf("broken body gave key %q", got)
	}
}
