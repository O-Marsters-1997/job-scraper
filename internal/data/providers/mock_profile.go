package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockProfileProvider lives outside _test.go so it can be imported by tests in other packages.
type MockProfileProvider struct {
	mu       sync.Mutex
	profiles map[string]dto.Profile // keyed by userID

	GetErr    error
	UpdateErr error
}

func NewMockProfileProvider() *MockProfileProvider {
	return &MockProfileProvider{profiles: make(map[string]dto.Profile)}
}

func (m *MockProfileProvider) Seed(userID string, p dto.Profile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[userID] = p
}

func (m *MockProfileProvider) GetProfile(_ context.Context, userID string) (dto.Profile, error) {
	if m.GetErr != nil {
		return dto.Profile{}, m.GetErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[userID]
	if !ok {
		return dto.Profile{}, fmt.Errorf("profile not found")
	}
	return p, nil
}

func (m *MockProfileProvider) UpdateEmail(_ context.Context, userID, email string) (dto.Profile, error) {
	if m.UpdateErr != nil {
		return dto.Profile{}, m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.profiles[userID]
	p.Email = email
	m.profiles[userID] = p
	return p, nil
}
