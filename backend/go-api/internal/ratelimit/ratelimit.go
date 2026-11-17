// Package ratelimit keeps one token bucket per key (an IP address, an email) in memory.
// Buckets live in this process only, which is right for a single API instance; several
// instances would each allow the full rate.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Limiter struct {
	mu        sync.Mutex
	every     time.Duration
	burst     int
	buckets   map[string]*bucket
	lastSweep time.Time
	now       func() time.Time
}

// New allows burst requests at once per key, then one more every interval.
func New(burst int, every time.Duration) *Limiter {
	return &Limiter{
		every:   every,
		burst:   burst,
		buckets: map[string]*bucket{},
		now:     time.Now,
	}
}

// Allow takes a token for key. When none is left it returns false and how long until one is.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{limiter: rate.NewLimiter(rate.Every(l.every), l.burst)}
		l.buckets[key] = b
	}
	b.lastSeen = now

	r := b.limiter.ReserveN(now, 1)
	if delay := r.DelayFrom(now); delay > 0 {
		r.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// sweep drops buckets idle long enough to have refilled completely, at most once a minute.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	full := l.every * time.Duration(l.burst)
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > full {
			delete(l.buckets, key)
		}
	}
}
