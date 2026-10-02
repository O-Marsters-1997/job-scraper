package sources

import (
	"errors"
	"math/rand/v2"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestGateWaitDrawsGapWithinJitterBounds(t *testing.T) {
	const gap = 2 * time.Millisecond
	now := time.Now()
	rnd := rand.New(rand.NewPCG(1, 2))
	g := NewGate(gap, 0)
	g.now = func() time.Time { return now }
	g.jitter = rnd.Float64
	prev := now
	for i := range 50 {
		if err := g.Wait(t.Context()); err != nil {
			t.Fatal(err)
		}
		if d := g.next.Sub(prev); d < gap/2 || d > gap*3/2 {
			t.Fatalf("wait %d drew a %v gap, want within [%v, %v]", i, d, gap/2, gap*3/2)
		}
		now, prev = g.next, g.next
	}
}

func TestGateWaitSpacesConcurrentCallers(t *testing.T) {
	const gap = 40 * time.Millisecond
	g := NewGate(gap, time.Hour)
	var (
		mu    sync.Mutex
		times []time.Time
		wg    sync.WaitGroup
	)
	for range 4 {
		wg.Go(func() {
			if err := g.Wait(t.Context()); err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			times = append(times, time.Now())
			mu.Unlock()
		})
	}
	wg.Wait()
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	for i := 1; i < len(times); i++ {
		if d := times[i].Sub(times[i-1]); d < gap/2-5*time.Millisecond {
			t.Errorf("callers %d and %d were %v apart, want >= %v", i-1, i, d, gap/2)
		}
	}
}

func TestGateWaitRefusesWhileBlocked(t *testing.T) {
	now := time.Now()
	g := NewGate(0, 0)
	g.now = func() time.Time { return now }
	g.Block(24 * time.Hour)
	if err := g.Wait(t.Context()); !errors.Is(err, ErrDeferred) {
		t.Errorf("Wait while blocked = %v, want ErrDeferred", err)
	}
	now = now.Add(25 * time.Hour)
	if err := g.Wait(t.Context()); err != nil {
		t.Errorf("Wait after the block lapsed = %v, want nil", err)
	}
}
