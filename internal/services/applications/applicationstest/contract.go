package applicationstest

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

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

func createApp(t *testing.T, f Fixture, in dto.CreateApplicationInput) dto.Application {
	t.Helper()
	in.JobID = f.JobID
	got, err := f.Store.CreateApplication(t.Context(), f.UserID, in)
	if err != nil {
		t.Fatalf("CreateApplication(%+v) err = %v", in, err)
	}
	return got
}

func createStatus(t *testing.T, f Fixture, name string) dto.ApplicationStatus {
	t.Helper()
	got, err := f.Store.CreateApplicationStatus(t.Context(), f.UserID, name, "#6366f1", nil)
	if err != nil {
		t.Fatalf("CreateApplicationStatus(%q) err = %v", name, err)
	}
	return got
}

// RunStoreContract proves newStore's applications.Store behaves the same
// whether it's the fake or the real store (ADR 0012). SeedDefaultStatuses is
// a tx-scoped port, covered in store/store_test.go.
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

	t.Run("create returns the application", func(t *testing.T) {
		f := newStore(t)
		got := createApp(t, f, dto.CreateApplicationInput{Notes: "referred by a friend"})
		want := dto.Application{UserID: f.UserID, JobID: f.JobID, Notes: "referred by a friend"}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.Application{}, "ID", "CreatedAt", "UpdatedAt")); diff != "" {
			t.Errorf("CreateApplication(...) mismatch (-want +got):\n%s", diff)
		}
		if got.ID == "" {
			t.Errorf("CreateApplication(...) ID is empty")
		}
	})

	t.Run("create keeps applied_at", func(t *testing.T) {
		f := newStore(t)
		got := createApp(t, f, dto.CreateApplicationInput{AppliedAt: new("2026-01-02")})
		want := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		if got.AppliedAt == nil || !got.AppliedAt.Equal(want) {
			t.Errorf("CreateApplication(applied_at=2026-01-02) AppliedAt = %v, want %v", got.AppliedAt, want)
		}
	})

	t.Run("list returns the user's applications", func(t *testing.T) {
		f := newStore(t)
		created := createApp(t, f, dto.CreateApplicationInput{})
		got, err := f.Store.ListApplications(t.Context(), f.UserID, "")
		if err != nil {
			t.Fatalf("ListApplications(...) err = %v", err)
		}
		if len(got) != 1 || got[0].ID != created.ID {
			t.Errorf("ListApplications(...) = %+v, want [%+v]", got, created)
		}
	})

	t.Run("update changes notes", func(t *testing.T) {
		f := newStore(t)
		created := createApp(t, f, dto.CreateApplicationInput{})
		got, err := f.Store.UpdateApplication(t.Context(), f.UserID, created.ID, dto.UpdateApplicationInput{Notes: "followed up"})
		if err != nil {
			t.Fatalf("UpdateApplication(...) err = %v", err)
		}
		if got.Notes != "followed up" {
			t.Errorf("UpdateApplication(...) notes = %q, want %q", got.Notes, "followed up")
		}
	})

	t.Run("update missing application returns not found", func(t *testing.T) {
		f := newStore(t)
		_, err := f.Store.UpdateApplication(t.Context(), f.UserID, missingID, dto.UpdateApplicationInput{})
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("UpdateApplication(missing) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete removes the application", func(t *testing.T) {
		f := newStore(t)
		created := createApp(t, f, dto.CreateApplicationInput{})
		if err := f.Store.DeleteApplication(t.Context(), f.UserID, created.ID); err != nil {
			t.Fatalf("DeleteApplication(...) err = %v", err)
		}
		got, err := f.Store.ListApplications(t.Context(), f.UserID, "")
		if err != nil {
			t.Fatalf("ListApplications(...) err = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("ListApplications after delete = %+v, want empty", got)
		}
	})

	t.Run("get applications for jobs keys by job id", func(t *testing.T) {
		f := newStore(t)
		created := createApp(t, f, dto.CreateApplicationInput{})
		got, err := f.Store.GetApplicationsForJobs(t.Context(), f.UserID, []string{f.JobID, missingID})
		if err != nil {
			t.Fatalf("GetApplicationsForJobs(...) err = %v", err)
		}
		want := map[string]dto.JobApplicationSummary{f.JobID: {ApplicationID: created.ID}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GetApplicationsForJobs(...) mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("create returns the status", func(t *testing.T) {
		f := newStore(t)
		got := createStatus(t, f, "Offer")
		want := dto.ApplicationStatus{UserID: f.UserID, Name: "Offer", Colour: "#6366f1"}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.ApplicationStatus{}, "ID", "CreatedAt")); diff != "" {
			t.Errorf("CreateApplicationStatus(...) mismatch (-want +got):\n%s", diff)
		}
		if got.ID == "" {
			t.Errorf("CreateApplicationStatus(...) ID is empty")
		}
	})

	t.Run("update changes name and colour", func(t *testing.T) {
		f := newStore(t)
		created := createStatus(t, f, "Offer")
		got, err := f.Store.UpdateApplicationStatus(t.Context(), created.ID, f.UserID, "Offer!", "#22c55e", nil)
		if err != nil {
			t.Fatalf("UpdateApplicationStatus(...) err = %v", err)
		}
		if got.Name != "Offer!" || got.Colour != "#22c55e" {
			t.Errorf("UpdateApplicationStatus(...) = %q/%q, want Offer!/#22c55e", got.Name, got.Colour)
		}
	})

	t.Run("update sets and clears the reply window", func(t *testing.T) {
		f := newStore(t)
		created := createStatus(t, f, "Applied")
		days := 7
		got, err := f.Store.UpdateApplicationStatus(t.Context(), created.ID, f.UserID, "Applied", "#6366f1", &days)
		if err != nil {
			t.Fatalf("UpdateApplicationStatus(set window) err = %v", err)
		}
		if got.ReplyWindowDays == nil || *got.ReplyWindowDays != days {
			t.Errorf("UpdateApplicationStatus(set window) ReplyWindowDays = %v, want %d", got.ReplyWindowDays, days)
		}
		got, err = f.Store.UpdateApplicationStatus(t.Context(), created.ID, f.UserID, "Applied", "#6366f1", nil)
		if err != nil {
			t.Fatalf("UpdateApplicationStatus(clear window) err = %v", err)
		}
		if got.ReplyWindowDays != nil {
			t.Errorf("UpdateApplicationStatus(clear window) ReplyWindowDays = %d, want nil", *got.ReplyWindowDays)
		}
	})

	t.Run("list returns the created status", func(t *testing.T) {
		f := newStore(t)
		created := createStatus(t, f, "Applied")
		got, err := f.Store.ListApplicationStatusesByUser(t.Context(), f.UserID)
		if err != nil {
			t.Fatalf("ListApplicationStatusesByUser(...) err = %v", err)
		}
		if len(got) != 1 || got[0].ID != created.ID {
			t.Errorf("ListApplicationStatusesByUser(...) = %+v, want [%+v]", got, created)
		}
	})

	t.Run("delete removes the status", func(t *testing.T) {
		f := newStore(t)
		created := createStatus(t, f, "Offer")
		if err := f.Store.DeleteApplicationStatus(t.Context(), created.ID, f.UserID); err != nil {
			t.Fatalf("DeleteApplicationStatus(...) err = %v", err)
		}
		got, err := f.Store.ListApplicationStatusesByUser(t.Context(), f.UserID)
		if err != nil {
			t.Fatalf("ListApplicationStatusesByUser(...) err = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("ListApplicationStatusesByUser after delete = %+v, want empty", got)
		}
	})

	t.Run("count for unused status is zero", func(t *testing.T) {
		f := newStore(t)
		created := createStatus(t, f, "Applied")
		count, err := f.Store.CountApplicationsUsingStatus(t.Context(), created.ID, f.UserID)
		if err != nil {
			t.Fatalf("CountApplicationsUsingStatus(unused) err = %v", err)
		}
		if count != 0 {
			t.Errorf("CountApplicationsUsingStatus(unused) = %d, want 0", count)
		}
	})

	t.Run("count for a status in use is the number of applications", func(t *testing.T) {
		f := newStore(t)
		created := createStatus(t, f, "Applied")
		createApp(t, f, dto.CreateApplicationInput{StatusID: created.ID})
		count, err := f.Store.CountApplicationsUsingStatus(t.Context(), created.ID, f.UserID)
		if err != nil {
			t.Fatalf("CountApplicationsUsingStatus(in use) err = %v", err)
		}
		if count != 1 {
			t.Errorf("CountApplicationsUsingStatus(in use) = %d, want 1", count)
		}
	})
}
