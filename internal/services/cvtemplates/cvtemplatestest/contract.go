package cvtemplatestest

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
)

// missingID is a well-formed but nonexistent UUID, valid input for both the
// fake and a real store that parses ids as UUIDs.
const missingID = "00000000-0000-0000-0000-000000000000"

// Fixture is what RunStoreContract needs: a store, a user, and a doc
// already tracked for them. cvtemplates.Store has no way to create a
// tracked doc itself; that's trackeddocs.Store.AddTrackedDoc.
type Fixture struct {
	Store        cvtemplates.Store
	UserID       string
	TrackedDocID string
}

// RunStoreContract proves newStore's cvtemplates.Store behaves the same
// whether it's the fake or the real store (ADR 0012).
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

	t.Run("list returns the tracked doc", func(t *testing.T) {
		f := newStore(t)
		got, err := f.Store.ListTrackedDocs(context.Background(), f.UserID)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, d := range got {
			if d.ID == f.TrackedDocID {
				found = true
			}
		}
		if !found {
			t.Fatalf("ListTrackedDocs(...) = %+v, want a doc with ID %q", got, f.TrackedDocID)
		}
	})

	t.Run("ensure tabs inserts default-visible rows", func(t *testing.T) {
		f := newStore(t)
		if err := f.Store.EnsureTabs(context.Background(), f.TrackedDocID, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListTabs(context.Background(), f.TrackedDocID)
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
		if err := f.Store.EnsureTabs(context.Background(), f.TrackedDocID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.EnsureTabs(context.Background(), f.TrackedDocID, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListTabs(context.Background(), f.TrackedDocID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("ListTabs(...) = %+v, want 2 rows", got)
		}
	})

	t.Run("ensure tabs is a no-op for an empty tab list", func(t *testing.T) {
		f := newStore(t)
		if err := f.Store.EnsureTabs(context.Background(), f.TrackedDocID, nil, nil); err != nil {
			t.Fatal(err)
		}
		got, err := f.Store.ListTabs(context.Background(), f.TrackedDocID)
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
