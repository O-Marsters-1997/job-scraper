package sources

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// ErrDeferred is returned by Gate.Wait, without sending a request, while the gate is blocked.
var ErrDeferred = errors.New("deferred while blocked")

const (
	defaultBlockedFor = 5 * time.Minute
	maxBlockedFor     = time.Hour
)

// DeferredError is the ErrDeferred a blocked Gate returns, carrying when the block lapses.
type DeferredError struct{ Until time.Time }

func (e *DeferredError) Error() string        { return ErrDeferred.Error() }
func (e *DeferredError) Is(target error) bool { return target == ErrDeferred }

// BlockedUntil reports whether err means the host is refusing us (a 429 or a blocked Gate)
// and when to try again, at most an hour out.
func BlockedUntil(err error, now time.Time) (time.Time, bool) {
	var deferred *DeferredError
	if errors.As(err, &deferred) {
		return now.Add(min(deferred.Until.Sub(now), maxBlockedFor)), true
	}
	var status *StatusError
	if errors.As(err, &status) && status.Code == http.StatusTooManyRequests {
		wait := status.RetryAfter
		if wait == 0 {
			wait = defaultBlockedFor
		}
		return now.Add(min(wait, maxBlockedFor)), true
	}
	return time.Time{}, false
}

// Gate spaces requests to one host by a random gap in [0.5*gap, 1.5*gap] and refuses them
// outright for a while after the host blocks us.
type Gate struct {
	mu           sync.Mutex
	gap          time.Duration
	blockFor     time.Duration
	now          func() time.Time
	jitter       func() float64
	next         time.Time
	blockedUntil time.Time
}

func NewGate(gap, blockFor time.Duration) *Gate {
	return &Gate{gap: gap, blockFor: blockFor, now: time.Now, jitter: rand.Float64}
}

// Wait blocks until the caller's slot, queueing behind earlier callers.
func (g *Gate) Wait(ctx context.Context) error {
	g.mu.Lock()
	now := g.now()
	if now.Before(g.blockedUntil) {
		until := g.blockedUntil
		g.mu.Unlock()
		return &DeferredError{Until: until}
	}
	slot := g.next
	if slot.Before(now) {
		slot = now
	}
	g.next = slot.Add(g.gap/2 + time.Duration(g.jitter()*float64(g.gap)))
	g.mu.Unlock()

	timer := time.NewTimer(slot.Sub(now))
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Block makes Wait return ErrDeferred for the next d, or for the gate's blockFor when d is zero.
func (g *Gate) Block(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if d == 0 {
		d = g.blockFor
	}
	g.blockedUntil = g.now().Add(d)
}
