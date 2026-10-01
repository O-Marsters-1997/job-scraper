package wttj

import (
	"errors"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestGateSpacesConcurrentRequests(t *testing.T) {
	const gap = 40 * time.Millisecond
	g := &gate{gap: gap, now: time.Now}
	var (
		mu    sync.Mutex
		times []time.Time
		wg    sync.WaitGroup
	)
	for range 4 {
		wg.Go(func() {
			if err := g.wait(t.Context()); err != nil {
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
		if d := times[i].Sub(times[i-1]); d < gap-5*time.Millisecond {
			t.Errorf("requests %d and %d were %v apart, want >= %v", i-1, i, d, gap)
		}
	}
}

func TestGateRefusesWhileBlocked(t *testing.T) {
	now := time.Now()
	g := &gate{now: func() time.Time { return now }}
	g.block(24 * time.Hour)
	if err := g.wait(t.Context()); !errors.Is(err, ErrDeferred) {
		t.Errorf("wait while blocked = %v, want ErrDeferred", err)
	}
	now = now.Add(25 * time.Hour)
	if err := g.wait(t.Context()); err != nil {
		t.Errorf("wait after the block lapsed = %v, want nil", err)
	}
}
