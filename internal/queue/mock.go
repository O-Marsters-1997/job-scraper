package queue

import (
	"context"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type MockQueue struct {
	mu               sync.Mutex
	jobs             []dto.QueuedJob
	tasks            []Task
	lastScraped      map[string]time.Time
	EnqueueErr       error
	EnqueueScrapeErr error
}

func NewMockQueue() *MockQueue { return &MockQueue{lastScraped: map[string]time.Time{}} }

func (m *MockQueue) EnqueueJobs(_ context.Context, jobs []dto.QueuedJob) error {
	if m.EnqueueErr != nil {
		return m.EnqueueErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs = append(m.jobs, jobs...)
	return nil
}

func (m *MockQueue) Publish(_ context.Context, task Task) error {
	if m.EnqueueScrapeErr != nil {
		return m.EnqueueScrapeErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, task)
	return nil
}

func (m *MockQueue) Jobs() []dto.QueuedJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]dto.QueuedJob(nil), m.jobs...)
}

func (m *MockQueue) Items() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.jobs))
	for i, job := range m.jobs {
		out[i] = job.URL
	}
	return out
}

func (m *MockQueue) Tasks() []Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Task(nil), m.tasks...)
}

func (m *MockQueue) ScrapeRequests() []dto.ScrapeRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.ScrapeRequest, len(m.tasks))
	for i, task := range m.tasks {
		out[i] = dto.ScrapeRequest{Target: dto.SourceTarget{ID: task.TargetID, RunID: task.RunID, Source: task.Source}}
	}
	return out
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
	last, ok := m.lastScraped[source]
	return last, ok, nil
}

func (m *MockQueue) SetLastScrapedAt(source string, when time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastScraped[source] = when
}
