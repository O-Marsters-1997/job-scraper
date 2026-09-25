package providers

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockJobScoreProvider lives outside _test.go so it can be imported by tests in other packages.
type MockJobScoreProvider struct {
	mu     sync.Mutex
	scores map[string]dto.JobScore

	GetErr    error
	UpsertErr error
}

func NewMockJobScoreProvider() *MockJobScoreProvider {
	return &MockJobScoreProvider{scores: make(map[string]dto.JobScore)}
}

func (m *MockJobScoreProvider) SetJobScore(js dto.JobScore) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scores[js.JobID] = js
}

func (m *MockJobScoreProvider) GetJobScore(_ context.Context, jobID, _ string) (dto.JobScore, error) {
	if m.GetErr != nil {
		return dto.JobScore{}, m.GetErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	js, ok := m.scores[jobID]
	if !ok {
		return dto.JobScore{}, ErrNotFound
	}
	return js, nil
}

func (m *MockJobScoreProvider) UpsertJobScoreSuitability(_ context.Context, jobID, userID string, score int, reasoning string, matched, missing []string) error {
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scores[jobID] = dto.JobScore{
		JobID:            jobID,
		UserID:           userID,
		SuitabilityScore: &score,
		Reasoning:        &reasoning,
		Matched:          matched,
		Missing:          missing,
	}
	return nil
}
