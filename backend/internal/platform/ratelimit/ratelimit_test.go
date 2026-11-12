package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeClock lets the test move time forward instead of sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newTestLimiter(limit int, window time.Duration) (*Limiter, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	l := New(limit, window)
	l.now = clock.now
	return l, clock
}

func TestAllow_BlocksAfterLimitUntilWindowEnds(t *testing.T) {
	l, clock := newTestLimiter(3, time.Minute)

	for i := range 3 {
		ok, _ := l.Allow("k")
		assert.True(t, ok, "attempt %d", i+1)
	}

	ok, retry := l.Allow("k")
	assert.False(t, ok)
	assert.Equal(t, time.Minute, retry)

	clock.t = clock.t.Add(40 * time.Second)
	ok, retry = l.Allow("k")
	assert.False(t, ok)
	assert.Equal(t, 20*time.Second, retry, "Retry-After counts down to the window end")

	clock.t = clock.t.Add(20 * time.Second)
	ok, _ = l.Allow("k")
	assert.True(t, ok, "a new window starts")
}

func TestAllow_KeysAreIndependent(t *testing.T) {
	l, _ := newTestLimiter(1, time.Minute)

	ok, _ := l.Allow("a")
	assert.True(t, ok)
	ok, _ = l.Allow("a")
	assert.False(t, ok)
	ok, _ = l.Allow("b")
	assert.True(t, ok, "another key has its own budget")
}

func TestReset(t *testing.T) {
	l, _ := newTestLimiter(1, time.Minute)

	l.Allow("k")
	ok, _ := l.Allow("k")
	assert.False(t, ok)

	l.Reset("k")
	ok, _ = l.Allow("k")
	assert.True(t, ok)
}
