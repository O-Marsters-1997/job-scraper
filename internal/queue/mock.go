package queue

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type MockQueue struct {
	mu               sync.Mutex
	tasks            []Task
	EnqueueScrapeErr error
}

func NewMockQueue() *MockQueue { return &MockQueue{} }

func (m *MockQueue) EnqueueJobs(context.Context, []dto.QueuedJob) error { return nil }

func (m *MockQueue) Publish(_ context.Context, task Task) error {
	if m.EnqueueScrapeErr != nil {
		return m.EnqueueScrapeErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, task)
	return nil
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
