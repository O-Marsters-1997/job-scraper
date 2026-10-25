package cvtemplatestest

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
)

const missingID = "00000000-0000-0000-0000-000000000000"

type Fixture struct {
	Store  cvtemplates.Store
	UserID string
	Other  string
}

// RunStoreContract proves newStore's cvtemplates.Store behaves the same
// whether it's the fake or the real store (ADR 0012).
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

	wantNotFound := func(t *testing.T, call string, err error) {
		t.Helper()
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("%s err = %v, want a not-found error", call, err)
		}
	}
	tabVisible := func(t *testing.T, st cvtemplates.Store, trackedDocID, tabID string) bool {
		t.Helper()
		tabs, err := st.ListTabs(t.Context(), trackedDocID)
		if err != nil {
			t.Fatalf("ListTabs(%q) err = %v", trackedDocID, err)
		}
		for _, tab := range tabs {
			if tab.TabID == tabID {
				return tab.Visible
			}
		}
		t.Fatalf("ListTabs(%q) = %+v, want tab %q", trackedDocID, tabs, tabID)
		return false
	}
	ensure := func(t *testing.T, st cvtemplates.Store, trackedDocID string, tabIDs ...string) {
		t.Helper()
		if err := st.EnsureTabs(t.Context(), trackedDocID, tabIDs, tabIDs); err != nil {
			t.Fatalf("EnsureTabs(%q, %v) err = %v", trackedDocID, tabIDs, err)
		}
	}
	setVisible := func(t *testing.T, st cvtemplates.Store, userID, docID, tabID string, visible bool) {
		t.Helper()
		if err := st.SetTabVisible(t.Context(), userID, docID, tabID, visible); err != nil {
			t.Fatalf("SetTabVisible(%q, %q, %q, %v) err = %v", userID, docID, tabID, visible, err)
		}
	}

	t.Run("add tracks the doc", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		Track(t, f.Store, f.UserID, "docA")
		if err := f.Store.RemoveTrackedDoc(t.Context(), f.UserID, "docA"); err != nil {
			t.Fatalf("RemoveTrackedDoc(docA) err = %v, want the doc to have been tracked", err)
		}
	})

	t.Run("add twice keeps one doc", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		first := Track(t, f.Store, f.UserID, "docA")
		if second := Track(t, f.Store, f.UserID, "docA"); second != first {
			t.Errorf("second Track = %q, want %q", second, first)
		}
		docs, err := f.Store.ListTrackedDocs(t.Context(), f.UserID, dto.TrackedDocCV)
		if err != nil {
			t.Fatalf("ListTrackedDocs err = %v", err)
		}
		if len(docs) != 1 {
			t.Errorf("ListTrackedDocs = %+v, want 1 doc", docs)
		}
	})

	t.Run("remove deletes the doc", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		Track(t, f.Store, f.UserID, "docA")
		if err := f.Store.RemoveTrackedDoc(t.Context(), f.UserID, "docA"); err != nil {
			t.Fatalf("RemoveTrackedDoc(docA) err = %v", err)
		}
		docs, err := f.Store.ListTrackedDocs(t.Context(), f.UserID, dto.TrackedDocCV)
		if err != nil {
			t.Fatalf("ListTrackedDocs err = %v", err)
		}
		if len(docs) != 0 {
			t.Errorf("ListTrackedDocs = %+v, want none", docs)
		}
	})

	t.Run("remove missing doc returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		err := f.Store.RemoveTrackedDoc(t.Context(), f.UserID, "no-such-doc")
		wantNotFound(t, "RemoveTrackedDoc(missing)", err)
	})

	t.Run("remove another user's doc returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		Track(t, f.Store, f.UserID, "docA")
		err := f.Store.RemoveTrackedDoc(t.Context(), f.Other, "docA")
		wantNotFound(t, "RemoveTrackedDoc(other user)", err)
	})

	t.Run("list returns only the user's docs", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		Track(t, f.Store, f.Other, "docB")
		got, err := f.Store.ListTrackedDocs(t.Context(), f.UserID, dto.TrackedDocCV)
		if err != nil {
			t.Fatalf("ListTrackedDocs err = %v", err)
		}
		if len(got) != 1 || got[0].ID != id {
			t.Fatalf("ListTrackedDocs = %+v, want only doc %q", got, id)
		}
	})

	t.Run("list filters by kind", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		cvID := TrackKind(t, f.Store, f.UserID, "docCV", dto.TrackedDocCV)
		letterID := TrackKind(t, f.Store, f.UserID, "docLetter", dto.TrackedDocLetter)
		for kind, wantID := range map[dto.TrackedDocKind]string{dto.TrackedDocCV: cvID, dto.TrackedDocLetter: letterID} {
			got, err := f.Store.ListTrackedDocs(t.Context(), f.UserID, kind)
			if err != nil {
				t.Fatalf("ListTrackedDocs(%q) err = %v", kind, err)
			}
			if len(got) != 1 || got[0].ID != wantID || got[0].Kind != kind {
				t.Errorf("ListTrackedDocs(%q) = %+v, want only doc %q", kind, got, wantID)
			}
		}
	})

	t.Run("hide missing tab returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		err := f.Store.SetTabVisible(t.Context(), f.UserID, "docA", "t-missing", false)
		wantNotFound(t, "SetTabVisible(missing, false)", err)
	})

	t.Run("show missing tab returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		err := f.Store.SetTabVisible(t.Context(), f.UserID, "docA", "t-missing", true)
		wantNotFound(t, "SetTabVisible(missing, true)", err)
	})

	t.Run("hide then show restores visibility", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		ensure(t, f.Store, id, "t1")

		setVisible(t, f.Store, f.UserID, "docA", "t1", false)
		if tabVisible(t, f.Store, id, "t1") {
			t.Error("tab t1 visible after hide, want hidden")
		}
		setVisible(t, f.Store, f.UserID, "docA", "t1", true)
		if !tabVisible(t, f.Store, id, "t1") {
			t.Error("tab t1 hidden after show, want visible")
		}
	})

	t.Run("another user cannot hide or show the tab", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		ensure(t, f.Store, id, "t1")

		for _, visible := range []bool{false, true} {
			err := f.Store.SetTabVisible(t.Context(), f.Other, "docA", "t1", visible)
			wantNotFound(t, "SetTabVisible(other user)", err)
		}
		if !tabVisible(t, f.Store, id, "t1") {
			t.Error("tab t1 hidden by another user, want visible")
		}
	})

	defaults := func(t *testing.T, st cvtemplates.Store, trackedDocID string) []string {
		t.Helper()
		tabs, err := st.ListTabs(t.Context(), trackedDocID)
		if err != nil {
			t.Fatalf("ListTabs(%q) err = %v", trackedDocID, err)
		}
		var got []string
		for _, tab := range tabs {
			if tab.IsDefault {
				got = append(got, tab.TabID)
			}
		}
		return got
	}
	setDefault := func(t *testing.T, st cvtemplates.Store, userID, docID, tabID string) {
		t.Helper()
		if err := st.SetTabDefault(t.Context(), userID, docID, tabID); err != nil {
			t.Fatalf("SetTabDefault(%q, %q, %q) err = %v", userID, docID, tabID, err)
		}
	}

	t.Run("setting a default moves it across docs", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		idA := Track(t, f.Store, f.UserID, "docA")
		idB := Track(t, f.Store, f.UserID, "docB")
		ensure(t, f.Store, idA, "t1", "t2")
		ensure(t, f.Store, idB, "t1")

		setDefault(t, f.Store, f.UserID, "docA", "t1")
		if got := defaults(t, f.Store, idA); len(got) != 1 || got[0] != "t1" {
			t.Fatalf("defaults(docA) = %v, want [t1]", got)
		}
		setDefault(t, f.Store, f.UserID, "docA", "t2")
		if got := defaults(t, f.Store, idA); len(got) != 1 || got[0] != "t2" {
			t.Errorf("defaults(docA) = %v, want [t2]", got)
		}
		setDefault(t, f.Store, f.UserID, "docB", "t1")
		if got := defaults(t, f.Store, idA); len(got) != 0 {
			t.Errorf("defaults(docA) = %v, want none after docB default", got)
		}
		if got := defaults(t, f.Store, idB); len(got) != 1 {
			t.Errorf("defaults(docB) = %v, want [t1]", got)
		}
	})

	t.Run("hiding the default clears it", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		ensure(t, f.Store, id, "t1")
		setDefault(t, f.Store, f.UserID, "docA", "t1")
		setVisible(t, f.Store, f.UserID, "docA", "t1", false)
		if got := defaults(t, f.Store, id); len(got) != 0 {
			t.Errorf("defaults = %v, want none after hide", got)
		}
	})

	t.Run("default is per user and rejects missing, hidden or foreign tabs", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		otherID := Track(t, f.Store, f.Other, "docB")
		ensure(t, f.Store, id, "t1", "t2")
		ensure(t, f.Store, otherID, "t1")
		setDefault(t, f.Store, f.UserID, "docA", "t1")
		setDefault(t, f.Store, f.Other, "docB", "t1")
		setVisible(t, f.Store, f.UserID, "docA", "t2", false)

		wantNotFound(t, "SetTabDefault(missing)", f.Store.SetTabDefault(t.Context(), f.UserID, "docA", "nope"))
		wantNotFound(t, "SetTabDefault(hidden)", f.Store.SetTabDefault(t.Context(), f.UserID, "docA", "t2"))
		wantNotFound(t, "SetTabDefault(foreign)", f.Store.SetTabDefault(t.Context(), f.UserID, "docB", "t1"))
		if got := defaults(t, f.Store, id); len(got) != 1 || got[0] != "t1" {
			t.Errorf("defaults(user) = %v, want [t1] kept after failed sets", got)
		}
		if got := defaults(t, f.Store, otherID); len(got) != 1 {
			t.Errorf("defaults(other) = %v, want [t1] untouched", got)
		}
	})

	t.Run("ensure tabs inserts default-visible rows", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		ensure(t, f.Store, id, "t1", "t2")
		for _, tabID := range []string{"t1", "t2"} {
			if !tabVisible(t, f.Store, id, tabID) {
				t.Errorf("tab %q hidden, want default visible", tabID)
			}
		}
	})

	t.Run("ensure tabs is idempotent", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		ensure(t, f.Store, id, "t1")
		ensure(t, f.Store, id, "t1", "t2")
		got, err := f.Store.ListTabs(t.Context(), id)
		if err != nil {
			t.Fatalf("ListTabs err = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("ListTabs = %+v, want 2 rows", got)
		}
	})

	t.Run("ensure tabs does not unhide a hidden tab", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		ensure(t, f.Store, id, "t1")
		setVisible(t, f.Store, f.UserID, "docA", "t1", false)
		ensure(t, f.Store, id, "t1")
		if tabVisible(t, f.Store, id, "t1") {
			t.Error("tab t1 visible after EnsureTabs, want still hidden")
		}
	})

	t.Run("ensure tabs is a no-op for an empty tab list", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		id := Track(t, f.Store, f.UserID, "docA")
		if err := f.Store.EnsureTabs(t.Context(), id, nil, nil); err != nil {
			t.Fatalf("EnsureTabs(nil) err = %v", err)
		}
		got, err := f.Store.ListTabs(t.Context(), id)
		if err != nil {
			t.Fatalf("ListTabs err = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("ListTabs = %+v, want empty", got)
		}
	})

	t.Run("list tabs is empty for an untracked doc", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		got, err := f.Store.ListTabs(t.Context(), missingID)
		if err != nil {
			t.Fatalf("ListTabs(missing) err = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("ListTabs(missing) = %+v, want empty", got)
		}
	})
}
