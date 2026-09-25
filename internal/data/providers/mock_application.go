package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockApplicationProvider lives outside _test.go so it can be imported by
// tests in other packages (the applications service tests in particular).
type MockApplicationProvider struct {
	mu   sync.Mutex
	apps map[string]dto.Application

	CreateErr error
	UpdateErr error
}

func NewMockApplicationProvider() *MockApplicationProvider {
	return &MockApplicationProvider{apps: make(map[string]dto.Application)}
}

func (m *MockApplicationProvider) CreateApplication(_ context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateErr != nil {
		return dto.Application{}, m.CreateErr
	}
	app := dto.Application{
		ID:         fmt.Sprintf("app-%d", len(m.apps)+1),
		UserID:     userID,
		JobID:      in.JobID,
		StatusID:   in.StatusID,
		Notes:      in.Notes,
		SalaryInfo: in.SalaryInfo,
	}
	m.apps[app.ID] = app
	return app, nil
}

func (m *MockApplicationProvider) ListApplicationsByUser(_ context.Context, userID string) ([]dto.ApplicationWithDetails, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range m.apps {
		if a.UserID == userID {
			out = append(out, dto.ApplicationWithDetails{ID: a.ID, UserID: a.UserID, JobID: a.JobID})
		}
	}
	return out, nil
}

func (m *MockApplicationProvider) ListApplicationsByUserAndStatus(_ context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range m.apps {
		if a.UserID == userID && a.StatusID == statusID {
			out = append(out, dto.ApplicationWithDetails{ID: a.ID, UserID: a.UserID, JobID: a.JobID, StatusID: a.StatusID})
		}
	}
	return out, nil
}

func (m *MockApplicationProvider) UpdateApplication(_ context.Context, userID, id string, in dto.UpdateApplicationInput) (dto.Application, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UpdateErr != nil {
		return dto.Application{}, m.UpdateErr
	}
	app, ok := m.apps[id]
	if !ok || app.UserID != userID {
		return dto.Application{}, ErrNotFound
	}
	app.StatusID = in.StatusID
	app.Notes = in.Notes
	app.SalaryInfo = in.SalaryInfo
	m.apps[id] = app
	return app, nil
}

func (m *MockApplicationProvider) DeleteApplication(_ context.Context, userID, id string) (struct{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.apps, id)
	return struct{}{}, nil
}

func (m *MockApplicationProvider) GetApplicationsForJobs(_ context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]dto.JobApplicationSummary{}
	for _, a := range m.apps {
		if a.UserID != userID {
			continue
		}
		for _, jobID := range jobIDs {
			if a.JobID == jobID {
				out[jobID] = dto.JobApplicationSummary{ApplicationID: a.ID, StatusID: a.StatusID}
			}
		}
	}
	return out, nil
}
