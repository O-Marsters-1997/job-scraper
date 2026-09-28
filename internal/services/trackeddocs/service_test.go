package trackeddocs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs/trackeddocstest"
)

type failingFileMeta struct{ err error }

func (f failingFileMeta) FileMeta(context.Context, string, string) (google.FileMeta, error) {
	return google.FileMeta{}, f.err
}

func TestAddDoc(t *testing.T) {
	t.Run("tracks a valid doc", func(t *testing.T) {
		gc := identitytest.NewDocsClient().WithDoc("abc1234567890", nil, google.FileMeta{})
		st := trackeddocstest.NewFakeStore()
		svc := trackeddocs.New(gc, st)

		if _, err := svc.AddDoc(context.Background(), "u1", dto.TrackedDocInput{URL: "https://docs.google.com/document/d/abc1234567890/edit"}); err != nil {
			t.Fatal(err)
		}
		if err := st.RemoveTrackedDoc(context.Background(), "u1", "abc1234567890"); err != nil {
			t.Errorf("doc should have been tracked: %v", err)
		}
	})

	t.Run("garbage URL", func(t *testing.T) {
		svc := trackeddocs.New(identitytest.NewDocsClient(), trackeddocstest.NewFakeStore())

		_, err := svc.AddDoc(context.Background(), "u1", dto.TrackedDocInput{URL: "not-a-url"})
		if !errors.Is(err, trackeddocs.ErrInvalidDoc) {
			t.Fatalf("err = %v, want ErrInvalidDoc", err)
		}
	})

	t.Run("inaccessible doc", func(t *testing.T) {
		gc := failingFileMeta{err: errors.New("permission denied")}
		svc := trackeddocs.New(gc, trackeddocstest.NewFakeStore())

		_, err := svc.AddDoc(context.Background(), "u1", dto.TrackedDocInput{URL: "https://docs.google.com/document/d/inaccessible123/edit"})
		if !errors.Is(err, trackeddocs.ErrInaccessibleDoc) {
			t.Fatalf("err = %v, want ErrInaccessibleDoc", err)
		}
	})
}

func TestRemoveDoc(t *testing.T) {
	t.Run("removes a tracked doc", func(t *testing.T) {
		st := trackeddocstest.NewFakeStore()
		if err := st.AddTrackedDoc(context.Background(), dto.AddTrackedDocInput{UserID: "u1", DocID: "docA"}); err != nil {
			t.Fatal(err)
		}
		svc := trackeddocs.New(identitytest.NewDocsClient(), st)

		if err := svc.RemoveDoc(context.Background(), "u1", "docA"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("missing doc returns not found", func(t *testing.T) {
		svc := trackeddocs.New(identitytest.NewDocsClient(), trackeddocstest.NewFakeStore())

		err := svc.RemoveDoc(context.Background(), "u1", "docA")
		status, ok := apperr.StatusFor(err)
		if !ok || status != apperr.KindNotFound.Status() {
			t.Errorf("expected a not-found error, got %v", err)
		}
	})
}

func TestHideTab(t *testing.T) {
	t.Run("hides a visible tab", func(t *testing.T) {
		st := trackeddocstest.NewFakeStore()
		st.SeedTab("u1", "docA", "t1", true)
		svc := trackeddocs.New(identitytest.NewDocsClient(), st)

		if _, err := svc.HideTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}); err != nil {
			t.Fatal(err)
		}
		if st.Visible("u1", "docA", "t1") {
			t.Error("tab should be hidden")
		}
	})

	t.Run("missing tab returns not found", func(t *testing.T) {
		svc := trackeddocs.New(identitytest.NewDocsClient(), trackeddocstest.NewFakeStore())

		_, err := svc.HideTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t-missing"})

		status, ok := apperr.StatusFor(err)
		if !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("expected a not-found error, got %v", err)
		}
	})
}

func TestShowTab(t *testing.T) {
	t.Run("shows a hidden tab", func(t *testing.T) {
		st := trackeddocstest.NewFakeStore()
		st.SeedTab("u1", "docA", "t1", false)
		svc := trackeddocs.New(identitytest.NewDocsClient(), st)

		if _, err := svc.ShowTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}); err != nil {
			t.Fatal(err)
		}
		if !st.Visible("u1", "docA", "t1") {
			t.Error("tab should be visible")
		}
	})

	t.Run("missing tab returns not found", func(t *testing.T) {
		svc := trackeddocs.New(identitytest.NewDocsClient(), trackeddocstest.NewFakeStore())

		_, err := svc.ShowTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t-missing"})

		status, ok := apperr.StatusFor(err)
		if !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("expected a not-found error, got %v", err)
		}
	})
}
