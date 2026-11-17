package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	now := time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)
	l := New(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := range 3 {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d within the burst was refused", i+1)
		}
	}
	ok, wait := l.Allow("a")
	if ok || wait <= 0 || wait > time.Minute {
		t.Fatalf("4th request: ok=%v wait=%v, want refused with a wait up to 1m", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("another key shares the first key's bucket")
	}

	now = now.Add(time.Minute)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("no token after one interval")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("refusals must not use up tokens, but a second token appeared after one interval")
	}
}

func TestSweepDropsIdleBuckets(t *testing.T) {
	now := time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)
	l := New(2, time.Minute)
	l.now = func() time.Time { return now }

	l.Allow("idle")
	now = now.Add(10 * time.Minute)
	l.Allow("fresh")

	if _, ok := l.buckets["idle"]; ok {
		t.Fatal("a bucket idle long enough to refill should have been dropped")
	}
	if _, ok := l.buckets["fresh"]; !ok {
		t.Fatal("the active bucket was dropped")
	}
}
