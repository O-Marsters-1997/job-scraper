package queue

import (
	"context"
	"sync"
	"time"
)

// FakeQueue is an in-memory implementation of JobQueue for use in tests.
// The fake lives outside _test.go so it can be imported by tests in other packages.
type FakeQueue struct {
	mu          sync.Mutex
	items       []string
	lastScraped map[string]time.Time

	// Injectable errors for failure-path tests.
	EnqueueErr error
	DequeueErr error
}

// NewFakeQueue returns a ready-to-use FakeQueue.
func NewFakeQueue() *FakeQueue {
	return &FakeQueue{lastScraped: make(map[string]time.Time)}
}

func (f *FakeQueue) Enqueue(_ context.Context, urls []string) error {
	if f.EnqueueErr != nil {
		return f.EnqueueErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append(f.items, urls...)
	return nil
}

func (f *FakeQueue) Dequeue(_ context.Context) (string, bool, error) {
	if f.DequeueErr != nil {
		return "", false, f.DequeueErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.items) == 0 {
		return "", false, nil
	}
	url := f.items[0]
	f.items = f.items[1:]
	return url, true, nil
}

func (f *FakeQueue) SetLastScraped(_ context.Context, source string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastScraped[source] = time.Now()
	return nil
}

func (f *FakeQueue) GetLastScraped(_ context.Context, source string) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.lastScraped[source]
	return t, ok, nil
}

func (f *FakeQueue) Close() {}

// Items returns a snapshot of the current queue contents (for test assertions).
func (f *FakeQueue) Items() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.items))
	copy(out, f.items)
	return out
}
