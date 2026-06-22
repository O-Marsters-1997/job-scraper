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
	scrapeReqs  []dto.ScrapeRequest
	lastScraped map[string]time.Time
	attempts    map[string]int
	deadLetter  []string

	EnqueueErr error
	DequeueErr error
	NackErr    error
}

func NewMockQueue() *MockQueue {
	return &MockQueue{
		lastScraped: make(map[string]time.Time),
		attempts:    make(map[string]int),
	}
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

func (m *MockQueue) Nack(_ context.Context, url string) error {
	if m.NackErr != nil {
		return m.NackErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts[url]++
	if m.attempts[url] >= maxAttempts {
		m.deadLetter = append(m.deadLetter, url)
		delete(m.attempts, url)
	}
	return nil
}

func (m *MockQueue) ClearAttempts(_ context.Context, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.attempts, url)
	return nil
}

func (m *MockQueue) EnqueueScrapeRequest(_ context.Context, req dto.ScrapeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scrapeReqs = append(m.scrapeReqs, req)
	return nil
}

func (m *MockQueue) DequeueScrapeRequest(_ context.Context) (dto.ScrapeRequest, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.scrapeReqs) == 0 {
		return dto.ScrapeRequest{}, false, nil
	}
	req := m.scrapeReqs[0]
	m.scrapeReqs = m.scrapeReqs[1:]
	return req, true, nil
}

func (m *MockQueue) ScrapeRequests() []dto.ScrapeRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.ScrapeRequest, len(m.scrapeReqs))
	copy(out, m.scrapeReqs)
	return out
}

func (m *MockQueue) Close() {}

func (m *MockQueue) DeadLetter() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.deadLetter))
	copy(out, m.deadLetter)
	return out
}

func (m *MockQueue) Attempts(url string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attempts[url]
}

func (m *MockQueue) SetLastScrapedAt(source string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastScraped[source] = t
}

func (m *MockQueue) Items() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.items))
	for i, j := range m.items {
		out[i] = j.URL
	}
	return out
}

func (m *MockQueue) Jobs() []dto.QueuedJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.QueuedJob, len(m.items))
	copy(out, m.items)
	return out
}
