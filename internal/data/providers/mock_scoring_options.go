package providers

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockScoringOptionsProvider lives outside _test.go so it can be imported by tests in other packages.
type MockScoringOptionsProvider struct {
	mu      sync.Mutex
	options []dto.ScoringOption

	ListErr error
}

func NewMockScoringOptionsProvider() *MockScoringOptionsProvider {
	return &MockScoringOptionsProvider{}
}

func (m *MockScoringOptionsProvider) Seed(options []dto.ScoringOption) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.options = options
}

func (m *MockScoringOptionsProvider) ListScoringOptions(_ context.Context) ([]dto.ScoringOption, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.ScoringOption, len(m.options))
	copy(out, m.options)
	return out, nil
}
