package applicationstest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
)

var defaultStatusNames = []string{"Saved", "Applied", "Interviewing", "Offer", "Rejected"}

type FakeStore struct {
	mu       sync.Mutex
	apps     map[string]dto.Application
	statuses map[string]dto.ApplicationStatus
}

func NewFakeStore() *FakeStore {
	return &FakeStore{apps: make(map[string]dto.Application), statuses: make(map[string]dto.ApplicationStatus)}
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
		AppliedAt:  parseDate(in.AppliedAt),
		SalaryInfo: in.SalaryInfo,
	}
	f.apps[app.ID] = app
	return app, nil
}

func (f *FakeStore) ListApplications(_ context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range f.apps {
		if a.UserID == userID && (statusID == "" || a.StatusID == statusID) {
			out = append(out, toDetails(a))
		}
	}
	return out, nil
}

func parseDate(s *string) *time.Time {
	if s == nil {
		return nil
	}
	d, err := time.Parse(time.DateOnly, *s)
	if err != nil {
		return nil
	}
	return &d
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
	app.AppliedAt = parseDate(in.AppliedAt)
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

func (f *FakeStore) CreateApplicationStatus(_ context.Context, userID, name, colour string) (dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := dto.ApplicationStatus{ID: fmt.Sprintf("status-%d", len(f.statuses)+1), UserID: userID, Name: name, Colour: colour}
	f.statuses[s.ID] = s
	return s, nil
}

func (f *FakeStore) UpdateApplicationStatus(_ context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.statuses[id]
	if !ok || s.UserID != userID {
		return dto.ApplicationStatus{}, apperr.NotFound("status not found")
	}
	s.Name, s.Colour = name, colour
	f.statuses[id] = s
	return s, nil
}

// DeleteApplicationStatus is a no-op for an unknown id, matching the real
// store's unconditional DELETE.
func (f *FakeStore) DeleteApplicationStatus(_ context.Context, id, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.statuses, id)
	return nil
}

func (f *FakeStore) CountApplicationsUsingStatus(_ context.Context, statusID, userID string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, a := range f.apps {
		if a.StatusID == statusID && a.UserID == userID {
			n++
		}
	}
	return n, nil
}

func (f *FakeStore) ListApplicationStatusesByUser(_ context.Context, userID string) ([]dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationStatus{}
	for _, s := range f.statuses {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *FakeStore) SeedDefaultStatuses(_ context.Context, _ pgx.Tx, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, name := range defaultStatusNames {
		s := dto.ApplicationStatus{ID: fmt.Sprintf("status-%d", len(f.statuses)+1), UserID: userID, Name: name}
		f.statuses[s.ID] = s
	}
	return nil
}

var _ applications.Store = (*FakeStore)(nil)
