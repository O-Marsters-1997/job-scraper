package applications_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/applications"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/fp"
)

func TestCreateRejectsInvalidAppliedAt(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	svc := applications.New(store)
	_, err := svc.Create(context.Background(), "user-1", dto.CreateApplicationInput{
		JobID:     "job-1",
		AppliedAt: fp.Some("not-a-date"),
	})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
	}
}

func TestCreateRequiresJobID(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	svc := applications.New(store)
	_, err := svc.Create(context.Background(), "user-1", dto.CreateApplicationInput{})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
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

func TestCreateSurfacesConflictFromProvider(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	store.CreateErr = providers.ErrApplicationExists
	svc := applications.New(store)
	_, err := svc.Create(context.Background(), "user-1", dto.CreateApplicationInput{JobID: "job-1"})
	if status, ok := apperr.StatusFor(err); !ok || status != 409 {
		t.Fatalf("status = %v, ok = %v, want 409", status, ok)
	}
}

func TestUpdateRejectsInvalidAppliedAt(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	svc := applications.New(store)
	_, err := svc.Update(context.Background(), "user-1", "app-1", dto.UpdateApplicationInput{
		AppliedAt: fp.Some("not-a-date"),
	})
	if status, ok := apperr.StatusFor(err); !ok || status != 400 {
		t.Fatalf("status = %v, ok = %v, want 400", status, ok)
	}
}

func TestUpdateSurfacesNotFoundFromProvider(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	svc := applications.New(store)
	_, err := svc.Update(context.Background(), "user-1", "missing", dto.UpdateApplicationInput{})
	if status, ok := apperr.StatusFor(err); !ok || status != 404 {
		t.Fatalf("status = %v, ok = %v, want 404", status, ok)
	}
}

func TestUpdateSucceeds(t *testing.T) {
	store := providers.NewMockApplicationProvider()
	created, err := store.CreateApplication(context.Background(), "user-1", dto.CreateApplicationInput{JobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	svc := applications.New(store)
	app, err := svc.Update(context.Background(), "user-1", created.ID, dto.UpdateApplicationInput{Notes: "followed up"})
	if err != nil {
		t.Fatal(err)
	}
	if app.Notes != "followed up" {
		t.Fatalf("notes = %q, want %q", app.Notes, "followed up")
	}
}
