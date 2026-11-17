package middleware

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"groundwork/internal/httpx"
	"groundwork/internal/logging"
	"groundwork/internal/ratelimit"
)

// ClientIP works out who is calling. Logins and registrations arrive from the frontend's
// server, not the browser, so the frontend forwards the visitor's address in X-Client-IP
// with X-Proxy-Secret; that address is used only when the secret matches. Otherwise it's
// the address of the connection: behind a proxy (Render), the last X-Forwarded-For entry,
// which the proxy itself appended, so a caller can't choose it. If that entry is one of the
// host's own addresses, direct callers share one bucket, which is stricter, never looser.
func ClientIP(proxySecret string, behindProxy bool) func(*http.Request) string {
	return func(r *http.Request) string {
		if proxySecret != "" {
			sent := r.Header.Get("X-Proxy-Secret")
			if subtle.ConstantTimeCompare([]byte(sent), []byte(proxySecret)) == 1 {
				if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Client-IP"))); ip != nil {
					return ip.String()
				}
			}
		}
		if behindProxy {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				if ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1])); ip != nil {
					return ip.String()
				}
			}
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
}

// RateLimit answers 429 with Retry-After once key's bucket is empty. An empty key isn't limited.
func RateLimit(limiter *ratelimit.Limiter, key func(*http.Request) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := key(r)
		if k == "" {
			next.ServeHTTP(w, r)
			return
		}
		if ok, wait := limiter.Allow(k); !ok {
			seconds := int(math.Ceil(wait.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			logging.From(r.Context()).Warn("rate limited", "path", r.URL.Path)
			httpx.WriteError(w, http.StatusTooManyRequests, fmt.Sprintf("Too many attempts. Try again in %s.", humanize(seconds)))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// EmailKey reads the "email" field of a JSON body, lower-cased, and puts the body back for
// the handler. Bodies it can't read give no key, and the handler rejects them as usual.
func EmailKey(r *http.Request) string {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var req struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(body, &req) != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(req.Email))
}

func humanize(seconds int) string {
	d := time.Duration(seconds) * time.Second
	switch {
	case d < time.Minute:
		return plural(seconds, "second")
	case d < time.Hour:
		return plural(int(math.Ceil(d.Minutes())), "minute")
	default:
		return plural(int(math.Ceil(d.Hours())), "hour")
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
