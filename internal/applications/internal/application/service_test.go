package application_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/applications/internal/application"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/fp"
)

type fakeStore struct {
	mu   sync.Mutex
	apps map[string]dto.Application

	createErr error
	updateErr error
}

func newFakeStore() *fakeStore { return &fakeStore{apps: make(map[string]dto.Application)} }

func (f *fakeStore) CreateApplication(_ context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return dto.Application{}, f.createErr
	}
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

func (f *fakeStore) ListApplicationsByUser(_ context.Context, userID string) ([]dto.ApplicationWithDetails, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range f.apps {
		if a.UserID == userID {
			out = append(out, dto.ApplicationWithDetails{ID: a.ID, UserID: a.UserID, JobID: a.JobID, StatusID: a.StatusID})
		}
	}
	return out, nil
}

func (f *fakeStore) ListApplicationsByUserAndStatus(_ context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range f.apps {
		if a.UserID == userID && a.StatusID == statusID {
			out = append(out, dto.ApplicationWithDetails{ID: a.ID, UserID: a.UserID, JobID: a.JobID, StatusID: a.StatusID})
		}
	}
	return out, nil
}

func (f *fakeStore) UpdateApplication(_ context.Context, userID, id string, in dto.UpdateApplicationInput) (dto.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return dto.Application{}, f.updateErr
	}
	app, ok := f.apps[id]
	if !ok || app.UserID != userID {
		return dto.Application{}, apperr.NotFound("not found")
	}
	app.StatusID = in.StatusID
	app.Notes = in.Notes
	app.SalaryInfo = in.SalaryInfo
	f.apps[id] = app
	return app, nil
}

func (f *fakeStore) DeleteApplication(_ context.Context, _, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.apps, id)
	return nil
}

func (f *fakeStore) GetApplicationsForJobs(_ context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]dto.JobApplicationSummary{}
	for _, a := range f.apps {
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

func TestCreate(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(store *fakeStore)
		in         dto.CreateApplicationInput
		wantStatus int
	}{
		{
			name:       "requires job id",
			in:         dto.CreateApplicationInput{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "rejects invalid applied_at",
			in: dto.CreateApplicationInput{
				JobID:     "job-1",
				AppliedAt: fp.Some("not-a-date"),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "surfaces conflict from store",
			setup: func(store *fakeStore) {
				store.createErr = apperr.Conflict("application already exists for this job")
			},
			in:         dto.CreateApplicationInput{JobID: "job-1"},
			wantStatus: http.StatusConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			svc := application.New(store)
			_, err := svc.Create(context.Background(), "user-1", tt.in)
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, tt.wantStatus)
			}
		})
	}
}

func TestCreateSucceeds(t *testing.T) {
	svc := application.New(newFakeStore())
	app, err := svc.Create(context.Background(), "user-1", dto.CreateApplicationInput{
		JobID:     "job-1",
		AppliedAt: fp.Some("2026-01-02"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.UserID != "user-1" || app.JobID != "job-1" {
		t.Fatalf("app = %+v, want user-1/job-1", app)
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		in         dto.UpdateApplicationInput
		wantStatus int
	}{
		{
			name:       "rejects invalid applied_at",
			id:         "app-1",
			in:         dto.UpdateApplicationInput{AppliedAt: fp.Some("not-a-date")},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "surfaces not found from store",
			id:         "missing",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := application.New(newFakeStore())
			tt.in.ID = tt.id
			_, err := svc.Update(context.Background(), "user-1", tt.in)
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, tt.wantStatus)
			}
		})
	}
}

func TestUpdateSucceeds(t *testing.T) {
	store := newFakeStore()
	created, err := store.CreateApplication(context.Background(), "user-1", dto.CreateApplicationInput{JobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(store)
	app, err := svc.Update(context.Background(), "user-1", dto.UpdateApplicationInput{ID: created.ID, Notes: "followed up"})
	if err != nil {
		t.Fatal(err)
	}
	if app.Notes != "followed up" {
		t.Fatalf("notes = %q, want %q", app.Notes, "followed up")
	}
}

func TestForJobsNoIDsReturnsEmptyMap(t *testing.T) {
	svc := application.New(newFakeStore())
	got, err := svc.ForJobs(context.Background(), "user-1", dto.ApplicationsForJobsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, want empty", got)
	}
}

func TestListFiltersByStatusWhenGiven(t *testing.T) {
	store := newFakeStore()
	if _, err := store.CreateApplication(context.Background(), "user-1", dto.CreateApplicationInput{JobID: "job-1"}); err != nil {
		t.Fatal(err)
	}
	svc := application.New(store)
	got, err := svc.List(context.Background(), "user-1", dto.ApplicationsQuery{StatusID: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, want empty", got)
	}
}
