package applications_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/services/applications"
	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/fp"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(store *providers.MockApplicationProvider)
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
			name: "surfaces conflict from provider",
			setup: func(store *providers.MockApplicationProvider) {
				store.CreateErr = providers.ErrApplicationExists
			},
			in:         dto.CreateApplicationInput{JobID: "job-1"},
			wantStatus: http.StatusConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockApplicationProvider()
			if tt.setup != nil {
				tt.setup(store)
			}
			svc := applications.New(store)
			_, err := svc.Create(context.Background(), "user-1", tt.in)
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, tt.wantStatus)
			}
		})
	}
}

func TestCreateSucceeds(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	svc := applications.New(store)
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
			name:       "surfaces not found from provider",
			id:         "missing",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providers.NewMockApplicationProvider()
			svc := applications.New(store)
			tt.in.ID = tt.id
			_, err := svc.Update(context.Background(), "user-1", tt.in)
			if status, ok := apperr.StatusFor(err); !ok || status != tt.wantStatus {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, tt.wantStatus)
			}
		})
	}
}

func TestUpdateSucceeds(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	created, err := store.CreateApplication(context.Background(), "user-1", dto.CreateApplicationInput{JobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	svc := applications.New(store)
	app, err := svc.Update(context.Background(), "user-1", dto.UpdateApplicationInput{ID: created.ID, Notes: "followed up"})
	if err != nil {
		t.Fatal(err)
	}
	if app.Notes != "followed up" {
		t.Fatalf("notes = %q, want %q", app.Notes, "followed up")
	}
}

func TestForJobsNoIDsReturnsEmptyMap(t *testing.T) {
	svc := applications.New(providers.NewMockApplicationProvider())
	got, err := svc.ForJobs(context.Background(), "user-1", dto.ApplicationsForJobsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, want empty", got)
	}
}

func TestListFiltersByStatusWhenGiven(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	if _, err := store.CreateApplication(context.Background(), "user-1", dto.CreateApplicationInput{JobID: "job-1"}); err != nil {
		t.Fatal(err)
	}
	svc := applications.New(store)
	got, err := svc.List(context.Background(), "user-1", dto.ApplicationsQuery{StatusID: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, want empty", got)
	}
}
