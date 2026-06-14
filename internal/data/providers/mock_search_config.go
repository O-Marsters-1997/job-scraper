package providers

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockSearchConfigProvider lives outside _test.go so it can be imported by tests in other packages.
type MockSearchConfigProvider struct {
	mu sync.Mutex
	// keyed by userID
	configs map[string]dto.SearchConfig

	GetErr    error
	UpsertErr error
}

func NewMockSearchConfigProvider() *MockSearchConfigProvider {
	return &MockSearchConfigProvider{configs: make(map[string]dto.SearchConfig)}
}

func (m *MockSearchConfigProvider) Seed(cfg dto.SearchConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[cfg.UserID] = cfg
}

func (m *MockSearchConfigProvider) GetSearchConfig(_ context.Context, userID string) (dto.SearchConfig, error) {
	if m.GetErr != nil {
		return dto.SearchConfig{}, m.GetErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configs[userID], nil
}

func (m *MockSearchConfigProvider) UpsertSearchConfig(_ context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	if m.UpsertErr != nil {
		return dto.SearchConfig{}, m.UpsertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[cfg.UserID] = cfg
	return cfg, nil
}
