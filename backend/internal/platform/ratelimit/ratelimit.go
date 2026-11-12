// Package ratelimit counts attempts per key in fixed time windows.
//
// Used to slow down password guessing on login and mass sign-ups.
//
// Trade-offs, chosen deliberately for this stage:
//   - In memory: counts are lost on restart and NOT shared between several API
//     instances. Fine for one instance; a shared store (PostgreSQL or Redis)
//     is needed once the API is scaled out.
//   - Fixed window: simple and predictable; allows up to 2x the limit right at a
//     window boundary, which is acceptable for brute-force protection.
package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	now    func() time.Time
	hits   map[string]*bucket
}

type bucket struct {
	count   int
	resetAt time.Time
}

// New allows `limit` attempts per key per `window`.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, now: time.Now, hits: make(map[string]*bucket)}
}

// Allow records one attempt for key. When the limit is exceeded it returns
// false and how long until the window resets (for the Retry-After header).
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.hits[key]
	if !ok || !now.Before(b.resetAt) {
		b = &bucket{resetAt: now.Add(l.window)}
		l.hits[key] = b
		l.sweep(now)
	}

	if b.count >= l.limit {
		return false, b.resetAt.Sub(now)
	}
	b.count++
	return true, 0
}

// Reset forgets a key, e.g. after a successful login, so only consecutive
// failures count toward the limit.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}

// sweep drops expired buckets so the map cannot grow without bound.
// Called only when a new bucket is created, so it costs nothing on the hot path.
func (l *Limiter) sweep(now time.Time) {
	if len(l.hits) < 1024 {
		return
	}
	for k, b := range l.hits {
		if !now.Before(b.resetAt) {
			delete(l.hits, k)
		}
	}
}
