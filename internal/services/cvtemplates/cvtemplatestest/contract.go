package cvtemplatestest

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
)

// missingID is a well-formed but nonexistent UUID, valid input for both the
// fake and a real store that parses ids as UUIDs.
const missingID = "00000000-0000-0000-0000-000000000000"

// Fixture is what RunStoreContract needs: a store and a user id it can
// track docs against. The fake accepts any string; a real store needs a
// row that user_id's foreign key resolves to.
type Fixture struct {
	Store  cvtemplates.Store
	UserID string
}

// RunStoreContract proves newStore's cvtemplates.Store behaves the same
// whether it's the fake or the real store (ADR 0012).
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

	track := func(t *testing.T, f Fixture, docID string) string {
		t.Helper()
		ctx := context.Background()
		if err := f.Store.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: f.UserID, DocID: docID}); err != nil {
			t.Fatal(err)
		}
		docs, err := f.Store.ListTrackedDocs(ctx, f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range docs {
			if d.DocID == docID {
				return d.ID
			}
		}
		t.Fatalf("ListTrackedDocs(...) = %+v, want a doc with DocID %q", docs, docID)
		return ""
	}

	wantNotFound := func(t *testing.T, call string, err error) {
		t.Helper()
		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("%s err = %v, want a not-found error", call, err)
		}
	}

	t.Run("add tracks the doc", func(t *testing.T) {
		f := newStore(t)
		if err := f.Store.AddTrackedDoc(context.Background(), dto.AddTrackedDocInput{UserID: f.UserID, DocID: "docA"}); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.RemoveTrackedDoc(context.Background(), f.UserID, "docA"); err != nil {
			t.Fatalf("doc should have been tracked: %v", err)
		}
	})

	t.Run("remove missing doc returns not found", func(t *testing.T) {
		f := newStore(t)
		err := f.Store.RemoveTrackedDoc(context.Background(), f.UserID, "no-such-doc")
		wantNotFound(t, "RemoveTrackedDoc(missing)", err)
	})

	t.Run("hide missing tab returns not found", func(t *testing.T) {
		f := newStore(t)
		err := f.Store.HideTab(context.Background(), f.UserID, "docA", "t-missing")
		wantNotFound(t, "HideTab(missing)", err)
	})

	t.Run("show missing tab returns not found", func(t *testing.T) {
		f := newStore(t)
		err := f.Store.ShowTab(context.Background(), f.UserID, "docA", "t-missing")
		wantNotFound(t, "ShowTab(missing)", err)
	})

	t.Run("list returns the tracked doc", func(t *testing.T) {
		f := newStore(t)
		id := track(t, f, "docA")
		got, err := f.Store.ListTrackedDocs(context.Background(), f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, d := range got {
			if d.ID == id {
				found = true
			}
		}
		if !found {
			t.Fatalf("ListTrackedDocs(...) = %+v, want a doc with ID %q", got, id)
		}
	})

	t.Run("ensure tabs inserts default-visible rows", func(t *testing.T) {
		f := newStore(t)
		id := track(t, f, "docA")
		if err := f.Store.EnsureTabs(context.Background(), id, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListTabs(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("ListTabs(...) = %+v, want 2 rows", got)
		}
		for _, tab := range got {
			if !tab.Visible {
				t.Errorf("tab %q should default to visible", tab.TabID)
			}
		}
	})

	t.Run("ensure tabs is idempotent", func(t *testing.T) {
		f := newStore(t)
		id := track(t, f, "docA")
		if err := f.Store.EnsureTabs(context.Background(), id, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.EnsureTabs(context.Background(), id, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListTabs(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("ListTabs(...) = %+v, want 2 rows", got)
		}
	})

	t.Run("ensure tabs is a no-op for an empty tab list", func(t *testing.T) {
		f := newStore(t)
		id := track(t, f, "docA")
		if err := f.Store.EnsureTabs(context.Background(), id, nil, nil); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListTabs(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("ListTabs(...) = %+v, want empty", got)
		}
	})

	t.Run("list tabs is empty for an untracked doc", func(t *testing.T) {
		f := newStore(t)
		got, err := f.Store.ListTabs(context.Background(), missingID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("ListTabs(missing) = %+v, want empty", got)
		}
	})
}
