package queue

import (
	"context"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockQueue lives outside _test.go so it can be imported by tests in other packages.
type MockQueue struct {
	mu          sync.Mutex
	items       []dto.QueuedJob
	lastScraped map[string]time.Time

	// Injectable errors for failure-path tests.
	EnqueueErr error
	DequeueErr error
}

func NewMockQueue() *MockQueue {
	return &MockQueue{lastScraped: make(map[string]time.Time)}
}

func (m *MockQueue) EnqueueJobs(_ context.Context, jobs []dto.QueuedJob) error {
	if m.EnqueueErr != nil {
		return m.EnqueueErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, jobs...)
	return nil
}

func (m *MockQueue) Dequeue(_ context.Context) (dto.QueuedJob, bool, error) {
	if m.DequeueErr != nil {
		return dto.QueuedJob{}, false, m.DequeueErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.items) == 0 {
		return dto.QueuedJob{}, false, nil
	}
	job := m.items[0]
	m.items = m.items[1:]
	return job, true, nil
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

// Items returns the URLs of all currently queued jobs.
func (m *MockQueue) Items() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.items))
	for i, j := range m.items {
		out[i] = j.URL
	}
	return out
}

// Jobs returns all currently queued jobs.
func (m *MockQueue) Jobs() []dto.QueuedJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.QueuedJob, len(m.items))
	copy(out, m.items)
	return out
}
