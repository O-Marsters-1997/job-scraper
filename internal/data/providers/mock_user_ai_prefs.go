package providers

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockUserAIPrefsProvider lives outside _test.go so it can be imported by tests in other packages.
type MockUserAIPrefsProvider struct {
	mu    sync.Mutex
	prefs map[string]dto.UserAIPrefs

	GetErr    error
	UpsertErr error
}

func NewMockUserAIPrefsProvider() *MockUserAIPrefsProvider {
	return &MockUserAIPrefsProvider{prefs: make(map[string]dto.UserAIPrefs)}
}

func (m *MockUserAIPrefsProvider) GetUserAIPrefs(_ context.Context, userID string) (dto.UserAIPrefs, error) {
	if m.GetErr != nil {
		return dto.UserAIPrefs{}, m.GetErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.prefs[userID]
	if !ok {
		return dto.UserAIPrefs{}, ErrNotFound
	}
	return p, nil
}

func (m *MockUserAIPrefsProvider) UpsertUserAIPrefs(_ context.Context, userID, suitabilityModel, reasoningModel string) (dto.UserAIPrefs, error) {
	if m.UpsertErr != nil {
		return dto.UserAIPrefs{}, m.UpsertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p := dto.UserAIPrefs{SuitabilityModel: suitabilityModel, ReasoningModel: reasoningModel}
	m.prefs[userID] = p
	return p, nil
}
