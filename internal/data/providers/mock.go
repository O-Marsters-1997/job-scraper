package providers

import (
	"context"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockJobProvider lives outside _test.go so it can be imported by tests in other packages.
type MockJobProvider struct {
	mu   sync.Mutex
	jobs map[string]dto.Job // keyed by URL

	// Injectable errors for failure-path tests.
	SaveErr    error
	NewURLsErr error
}

func NewMockJobProvider() *MockJobProvider {
	return &MockJobProvider{jobs: make(map[string]dto.Job)}
}

func (m *MockJobProvider) Save(_ context.Context, jobs []dto.Job) ([]dto.Job, error) {
	if m.SaveErr != nil {
		return nil, m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range jobs {
		m.jobs[j.URL] = j
	}
	return jobs, nil
}

func (m *MockJobProvider) NewURLs(_ context.Context, urls []string) ([]string, error) {
	if m.NewURLsErr != nil {
		return nil, m.NewURLsErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, u := range urls {
		if _, exists := m.jobs[u]; !exists {
			out = append(out, u)
		}
	}
	return out, nil
}

func (m *MockJobProvider) List(_ context.Context, _ string) ([]dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out, nil
}

func (m *MockJobProvider) GetJob(_ context.Context, jobID, _ string) (dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if j.ID == jobID {
			return j, nil
		}
	}
	return dto.Job{}, ErrNotFound
}

func (m *MockJobProvider) ListSince(_ context.Context, since time.Time) ([]dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []dto.Job
	for _, j := range m.jobs {
		if j.ScrapedAt.After(since) {
			out = append(out, j)
		}
	}
	return out, nil
}

func (m *MockJobProvider) Jobs() []dto.Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out
}
