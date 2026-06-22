package providers

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ErrDuplicateSourceTarget is returned by CreateSourceTarget when the (user_id, source, value, filters) tuple already exists.
var ErrDuplicateSourceTarget = errors.New("source target already exists")

type MockSourceTargetProvider struct {
	mu      sync.Mutex
	targets []dto.SourceTarget
	nextID  int

	CreateErr error
	UpdateErr error
	DeleteErr error
}

func NewMockSourceTargetProvider() *MockSourceTargetProvider {
	return &MockSourceTargetProvider{nextID: 1}
}

func (m *MockSourceTargetProvider) ListSourceTargetsByUser(_ context.Context, userID string) ([]dto.SourceTarget, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []dto.SourceTarget
	for _, t := range m.targets {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *MockSourceTargetProvider) ListEnabledSourceTargets(_ context.Context) ([]dto.SourceTarget, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []dto.SourceTarget
	for _, t := range m.targets {
		if t.Enabled {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *MockSourceTargetProvider) CreateSourceTarget(_ context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error) {
	if m.CreateErr != nil {
		return dto.SourceTarget{}, m.CreateErr
	}
	if filters == nil {
		filters = map[string]string{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.targets {
		if t.UserID == userID && t.Source == source && t.Value == value && mapsEqual(t.Filters, filters) {
			return dto.SourceTarget{}, ErrDuplicateSourceTarget
		}
	}
	t := dto.SourceTarget{
		ID:      fmt.Sprintf("target-%d", m.nextID),
		UserID:  userID,
		Source:  source,
		Value:   value,
		Enabled: enabled,
		Filters: filters,
	}
	m.nextID++
	m.targets = append(m.targets, t)
	return t, nil
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (m *MockSourceTargetProvider) UpdateSourceTarget(_ context.Context, id, userID string, enabled bool) (dto.SourceTarget, error) {
	if m.UpdateErr != nil {
		return dto.SourceTarget{}, m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.targets {
		if t.ID == id && t.UserID == userID {
			m.targets[i].Enabled = enabled
			return m.targets[i], nil
		}
	}
	return dto.SourceTarget{}, ErrNotFound
}

func (m *MockSourceTargetProvider) DeleteSourceTarget(_ context.Context, id, userID string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.targets {
		if t.ID == id && t.UserID == userID {
			m.targets = append(m.targets[:i], m.targets[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
