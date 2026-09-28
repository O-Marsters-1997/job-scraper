// Package applicationstest is the applications feature's test double: a
// map-backed fake of applications.Store, proven against the real store by
// RunStoreContract (ADR 0012).
package applicationstest

import (
	"context"
	"fmt"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
)

type FakeStore struct {
	mu   sync.Mutex
	apps map[string]dto.Application
}

func NewFakeStore() *FakeStore {
	return &FakeStore{apps: make(map[string]dto.Application)}
}

func (f *FakeStore) CreateApplication(_ context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	app := dto.Application{
		ID:         fmt.Sprintf("app-%d", len(f.apps)+1),
		UserID:     userID,
		JobID:      in.JobID,
		StatusID:   in.StatusID,
		Notes:      in.Notes,
		SalaryInfo: in.SalaryInfo,
	}
	f.apps[app.ID] = app
	return app, nil
}

func (f *FakeStore) ListApplicationsByUser(_ context.Context, userID string) ([]dto.ApplicationWithDetails, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range f.apps {
		if a.UserID == userID {
			out = append(out, toDetails(a))
		}
	}
	return out, nil
}

func (f *FakeStore) ListApplicationsByUserAndStatus(_ context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range f.apps {
		if a.UserID == userID && a.StatusID == statusID {
			out = append(out, toDetails(a))
		}
	}
	return out, nil
}

func toDetails(a dto.Application) dto.ApplicationWithDetails {
	return dto.ApplicationWithDetails{
		ID: a.ID, UserID: a.UserID, JobID: a.JobID, StatusID: a.StatusID,
		Notes: a.Notes, AppliedAt: a.AppliedAt, SalaryInfo: a.SalaryInfo,
	}
}

func (f *FakeStore) UpdateApplication(_ context.Context, userID, id string, in dto.UpdateApplicationInput) (dto.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	app, ok := f.apps[id]
	if !ok || app.UserID != userID {
		return dto.Application{}, data.ErrNotFound
	}
	app.StatusID = in.StatusID
	app.Notes = in.Notes
	app.SalaryInfo = in.SalaryInfo
	f.apps[id] = app
	return app, nil
}

// DeleteApplication is a no-op for an unknown id, matching the real store's
// unconditional DELETE.
func (f *FakeStore) DeleteApplication(_ context.Context, _, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.apps, id)
	return nil
}

func (f *FakeStore) GetApplicationsForJobs(_ context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	wanted := make(map[string]bool, len(jobIDs))
	for _, id := range jobIDs {
		wanted[id] = true
	}
	out := map[string]dto.JobApplicationSummary{}
	for _, a := range f.apps {
		if a.UserID == userID && wanted[a.JobID] {
			out[a.JobID] = dto.JobApplicationSummary{ApplicationID: a.ID, StatusID: a.StatusID}
		}
	}
	return out, nil
}

var _ applications.Store = (*FakeStore)(nil)
