package queue

import (
	"context"
	"sync"
	"time"
)

// MockQueue lives outside _test.go so it can be imported by tests in other packages.
type MockQueue struct {
	mu          sync.Mutex
	items       []string
	lastScraped map[string]time.Time

	// Injectable errors for failure-path tests.
	EnqueueErr error
	DequeueErr error
}

func NewMockQueue() *MockQueue {
	return &MockQueue{lastScraped: make(map[string]time.Time)}
}

func (m *MockQueue) Enqueue(_ context.Context, urls []string) error {
	if m.EnqueueErr != nil {
		return m.EnqueueErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, urls...)
	return nil
}

func (m *MockQueue) Dequeue(_ context.Context) (string, bool, error) {
	if m.DequeueErr != nil {
		return "", false, m.DequeueErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.items) == 0 {
		return "", false, nil
	}
	url := m.items[0]
	m.items = m.items[1:]
	return url, true, nil
}

func (m *MockQueue) SetLastScraped(_ context.Context, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastScraped[source] = time.Now()
	return nil
}

func (m *MockQueue) GetLastScraped(_ context.Context, source string) (time.Time, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.lastScraped[source]
	return t, ok, nil
}

func (m *MockQueue) Close() {}

func (m *MockQueue) SetLastScrapedAt(source string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastScraped[source] = t
}

func (m *MockQueue) Items() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.items))
	copy(out, m.items)
	return out
}
