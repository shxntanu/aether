package httpapi

import (
	"sync"
	"time"
)

const limiterCapacity = 10_000

// Limiter applies a fixed-window rate limit in local process memory.
//
// State is not shared across processes, so clustered deployments must enforce
// any global limits at a separate boundary.
type Limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	entries map[string]limiterEntry
}

type limiterEntry struct {
	count     int
	expiresAt time.Time
}

// NewLimiter constructs a fixed-window limiter keyed by opaque caller hashes.
func NewLimiter(limit int, window time.Duration, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{
		limit:   limit,
		window:  window,
		now:     now,
		entries: make(map[string]limiterEntry),
	}
}

// Allow records one request for key and reports whether it fits the limit.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}
	if key == "" || l.limit <= 0 || l.window <= 0 {
		return false, l.window
	}

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for existingKey, entry := range l.entries {
		if !entry.expiresAt.After(now) {
			delete(l.entries, existingKey)
		}
	}

	entry, found := l.entries[key]
	if !found {
		if len(l.entries) >= limiterCapacity {
			return false, l.window
		}
		l.entries[key] = limiterEntry{
			count:     1,
			expiresAt: now.Add(l.window),
		}
		return true, 0
	}

	if entry.count >= l.limit {
		retryAfter := entry.expiresAt.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	entry.count++
	l.entries[key] = entry
	return true, 0
}
