package trackeddocstest

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs"
)

// Fixture is what RunStoreContract needs: a store and a user id it can
// track docs against. The fake accepts any string; a real store needs a
// row that user_id's foreign key resolves to.
type Fixture struct {
	Store  trackeddocs.Store
	UserID string
}

// RunStoreContract proves newStore's trackeddocs.Store behaves the same
// whether it's the fake or the real store (ADR 0012).
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

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
		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("RemoveTrackedDoc(missing) err = %v, want a not-found error", err)
		}
	})

	t.Run("hide missing tab returns not found", func(t *testing.T) {
		f := newStore(t)
		err := f.Store.HideTab(context.Background(), f.UserID, "docA", "t-missing")
		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("HideTab(missing) err = %v, want a not-found error", err)
		}
	})

	t.Run("show missing tab returns not found", func(t *testing.T) {
		f := newStore(t)
		err := f.Store.ShowTab(context.Background(), f.UserID, "docA", "t-missing")
		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("ShowTab(missing) err = %v, want a not-found error", err)
		}
	})
}
