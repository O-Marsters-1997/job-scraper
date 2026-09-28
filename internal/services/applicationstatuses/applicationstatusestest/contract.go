package applicationstatusestest

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses"
)

// Fixture is what RunStoreContract needs: a store and a user it can create
// statuses against. The fake accepts any string; a real store needs a row
// that user_id's foreign key resolves to.
type Fixture struct {
	Store  applicationstatuses.Store
	UserID string
}

// RunStoreContract proves newStore's applicationstatuses.Store behaves the
// same whether it's the fake or the real store (ADR 0012). SeedDefaultStatuses
// is a tx-scoped port, covered separately in store/store_test.go.
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

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
}
