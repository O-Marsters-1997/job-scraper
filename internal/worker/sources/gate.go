package sources

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"
)

// ErrDeferred is returned by Gate.Wait, without sending a request, while the gate is blocked.
var ErrDeferred = errors.New("deferred while blocked")

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
		g.mu.Unlock()
		return ErrDeferred
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
