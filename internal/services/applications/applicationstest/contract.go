package applicationstest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
)

// missingID is a well-formed but nonexistent UUID, valid input for both the
// fake and a real store that parses ids as UUIDs.
const missingID = "00000000-0000-0000-0000-000000000000"

type Fixture struct {
	Store  applications.Store
	UserID string
	JobID  string
}

// RunStoreContract proves newStore's applications.Store behaves the same
// whether it's the fake or the real store (ADR 0012). SeedDefaultStatuses is
// a tx-scoped port, covered in store/store_test.go.
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

	t.Run("create returns the application", func(t *testing.T) {
		f := newStore(t)
		got, err := f.Store.CreateApplication(context.Background(), f.UserID, dto.CreateApplicationInput{
			JobID:     f.JobID,
			Notes:     "referred by a friend",
			AppliedAt: new("2026-01-02"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.ID == "" || got.UserID != f.UserID || got.JobID != f.JobID || got.Notes != "referred by a friend" {
			t.Fatalf("CreateApplication(...) = %+v", got)
		}
	})

	t.Run("list returns the user's applications", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplication(context.Background(), f.UserID, dto.CreateApplicationInput{JobID: f.JobID})
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListApplications(context.Background(), f.UserID, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != created.ID {
			t.Fatalf("ListApplications(...) = %+v, want [%+v]", got, created)
		}
	})

	t.Run("update changes notes", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplication(context.Background(), f.UserID, dto.CreateApplicationInput{JobID: f.JobID})
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.UpdateApplication(context.Background(), f.UserID, created.ID, dto.UpdateApplicationInput{Notes: "followed up"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Notes != "followed up" {
			t.Fatalf("UpdateApplication(...) notes = %q, want %q", got.Notes, "followed up")
		}
	})

	t.Run("update missing application returns not found", func(t *testing.T) {
		f := newStore(t)
		_, err := f.Store.UpdateApplication(context.Background(), f.UserID, missingID, dto.UpdateApplicationInput{})
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("UpdateApplication(missing) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete removes the application", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplication(context.Background(), f.UserID, dto.CreateApplicationInput{JobID: f.JobID})
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Store.DeleteApplication(context.Background(), f.UserID, created.ID); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListApplications(context.Background(), f.UserID, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("ListApplications after delete = %+v, want empty", got)
		}
	})

	t.Run("get applications for jobs keys by job id", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplication(context.Background(), f.UserID, dto.CreateApplicationInput{JobID: f.JobID})
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.GetApplicationsForJobs(context.Background(), f.UserID, []string{f.JobID, missingID})
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]dto.JobApplicationSummary{f.JobID: {ApplicationID: created.ID}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("GetApplicationsForJobs(...) mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("create returns the status", func(t *testing.T) {
		f := newStore(t)
		got, err := f.Store.CreateApplicationStatus(context.Background(), f.UserID, "Offer", "#22c55e")
		if err != nil {
			t.Fatal(err)
		}
		if got.ID == "" || got.Name != "Offer" || got.Colour != "#22c55e" {
			t.Fatalf("CreateApplicationStatus(...) = %+v", got)
		}
	})

	t.Run("update changes name and colour", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplicationStatus(context.Background(), f.UserID, "Offer", "#22c55e")
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.UpdateApplicationStatus(context.Background(), created.ID, f.UserID, "Offer!", "#22c55e")
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Offer!" {
			t.Fatalf("UpdateApplicationStatus(...) name = %q, want %q", got.Name, "Offer!")
		}
	})

	t.Run("list returns the created status", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplicationStatus(context.Background(), f.UserID, "Applied", "#6366f1")
		if err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListApplicationStatusesByUser(context.Background(), f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != created.ID {
			t.Fatalf("ListApplicationStatusesByUser(...) = %+v, want [%+v]", got, created)
		}
	})

	t.Run("delete removes the status", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplicationStatus(context.Background(), f.UserID, "Offer", "#22c55e")
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Store.DeleteApplicationStatus(context.Background(), created.ID, f.UserID); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListApplicationStatusesByUser(context.Background(), f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("ListApplicationStatusesByUser after delete = %+v, want empty", got)
		}
	})

	t.Run("count for unused status is zero", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplicationStatus(context.Background(), f.UserID, "Applied", "#6366f1")
		if err != nil {
			t.Fatal(err)
		}
		count, err := f.Store.CountApplicationsUsingStatus(context.Background(), created.ID, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("CountApplicationsUsingStatus(unused) = %d, want 0", count)
		}
	})

	t.Run("count for a status in use is the number of applications", func(t *testing.T) {
		f := newStore(t)
		created, err := f.Store.CreateApplicationStatus(context.Background(), f.UserID, "Applied", "#6366f1")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Store.CreateApplication(context.Background(), f.UserID, dto.CreateApplicationInput{JobID: f.JobID, StatusID: created.ID}); err != nil {
			t.Fatal(err)
		}
		count, err := f.Store.CountApplicationsUsingStatus(context.Background(), created.ID, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("CountApplicationsUsingStatus(in use) = %d, want 1", count)
		}
	})
}
