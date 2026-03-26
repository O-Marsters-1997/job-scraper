package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockApplicationStatusProvider is a no-op mock for use in tests.
type MockApplicationStatusProvider struct{}

func (m *MockApplicationStatusProvider) SeedDefaultStatuses(_ context.Context, _ string) error {
	return nil
}
func (m *MockApplicationStatusProvider) CreateApplicationStatus(_ context.Context, _, _, _ string) (dto.ApplicationStatus, error) {
	return dto.ApplicationStatus{}, nil
}
func (m *MockApplicationStatusProvider) ListApplicationStatusesByUser(_ context.Context, _ string) ([]dto.ApplicationStatus, error) {
	return nil, nil
}
func (m *MockApplicationStatusProvider) UpdateApplicationStatus(_ context.Context, _, _, _, _ string) (dto.ApplicationStatus, error) {
	return dto.ApplicationStatus{}, nil
}
func (m *MockApplicationStatusProvider) DeleteApplicationStatus(_ context.Context, _, _ string) error {
	return nil
}
func (m *MockApplicationStatusProvider) CountApplicationsUsingStatus(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
}

// MockUserProvider lives outside _test.go so it can be imported by tests in other packages.
type MockUserProvider struct {
	mu    sync.Mutex
	users map[string]dto.User // keyed by username

	GetUserErr error
	CreateErr  error
}

func NewMockUserProvider() *MockUserProvider {
	return &MockUserProvider{users: make(map[string]dto.User)}
}

func (m *MockUserProvider) Seed(u dto.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.Username] = u
}

func (m *MockUserProvider) GetUserByUsername(_ context.Context, username string) (dto.User, error) {
	if m.GetUserErr != nil {
		return dto.User{}, m.GetUserErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[username]
	if !ok {
		return dto.User{}, fmt.Errorf("user not found")
	}
	return u, nil
}

func (m *MockUserProvider) CreateUser(_ context.Context, username, passwordHash string) (dto.User, error) {
	if m.CreateErr != nil {
		return dto.User{}, m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u := dto.User{ID: username + "-id", Username: username, PasswordHash: passwordHash}
	m.users[username] = u
	return u, nil
}

// MockSessionProvider lives outside _test.go so it can be imported by tests in other packages.
type MockSessionProvider struct {
	mu       sync.Mutex
	sessions map[string]dto.Session // keyed by session id

	CreateErr error
	GetErr    error
	DeleteErr error
}

func NewMockSessionProvider() *MockSessionProvider {
	return &MockSessionProvider{sessions: make(map[string]dto.Session)}
}

func (m *MockSessionProvider) Seed(s dto.Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
}

func (m *MockSessionProvider) CreateSession(_ context.Context, userID string, expiresAt time.Time) (dto.Session, error) {
	if m.CreateErr != nil {
		return dto.Session{}, m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("session-%d", len(m.sessions))
	s := dto.Session{ID: id, UserID: userID, ExpiresAt: expiresAt}
	m.sessions[id] = s
	return s, nil
}

func (m *MockSessionProvider) GetSession(_ context.Context, id string) (dto.Session, error) {
	if m.GetErr != nil {
		return dto.Session{}, m.GetErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return dto.Session{}, fmt.Errorf("session not found")
	}
	if time.Now().After(s.ExpiresAt) {
		return dto.Session{}, fmt.Errorf("session expired")
	}
	return s, nil
}

func (m *MockSessionProvider) DeleteSession(_ context.Context, id string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *MockSessionProvider) DeleteExpiredSessions(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		if time.Now().After(s.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
	return nil
}
