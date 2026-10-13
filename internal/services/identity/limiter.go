package identity

import (
	"sync"
	"time"
)

const (
	userAttemptEvery = 6 * time.Second
	userAttemptBurst = 5
	ipAttemptEvery   = 2 * time.Second
	ipAttemptBurst   = 30
	limiterSweepAt   = 1024
)

type bucket struct {
	tokens float64
	at     time.Time
}

type attemptLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	every   time.Duration
	burst   float64
	buckets map[string]*bucket
}

func newAttemptLimiter(now func() time.Time, every time.Duration, burst int) *attemptLimiter {
	return &attemptLimiter{now: now, every: every, burst: float64(burst), buckets: map[string]*bucket{}}
}

func (l *attemptLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.buckets) >= limiterSweepAt {
		l.sweep(now)
	}
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, at: now}
		l.buckets[key] = b
	}
	b.tokens = l.refilled(b, now)
	b.at = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *attemptLimiter) refilled(b *bucket, now time.Time) float64 {
	return min(l.burst, b.tokens+float64(now.Sub(b.at))/float64(l.every))
}

func (l *attemptLimiter) sweep(now time.Time) {
	for k, b := range l.buckets {
		if l.refilled(b, now) >= l.burst {
			delete(l.buckets, k)
		}
	}
}
