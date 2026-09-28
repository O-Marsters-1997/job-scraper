package store_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

var seedCounter atomic.Int64

func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	username := fmt.Sprintf("user-%s-%d", t.Name(), seedCounter.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`,
		username).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func seedTrackedDoc(t *testing.T, st *store.Store, pool *pgxpool.Pool, docID string) (userID, trackedDocID string) {
	t.Helper()
	ctx := context.Background()
	userID = insertUser(t, pool)
	if err := st.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID}); err != nil {
		t.Fatalf("AddTrackedDoc: %v", err)
	}
	docs, err := st.ListTrackedDocs(ctx, userID)
	if err != nil {
		t.Fatalf("ListTrackedDocs: %v", err)
	}
	for _, d := range docs {
		if d.DocID == docID {
			return userID, d.ID
		}
	}
	t.Fatalf("seeded doc %q not found after insert", docID)
	return "", ""
}

func TestStoreContract(t *testing.T) {
	cvtemplatestest.RunStoreContract(t, func(t *testing.T) cvtemplatestest.Fixture {
		t.Helper()
		st, pool := newStore(t)
		return cvtemplatestest.Fixture{Store: st, UserID: insertUser(t, pool)}
	})
}

func TestAddTrackedDoc_DuplicateIsNoOp(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)
	ctx := context.Background()

	if err := st.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: "docA"}); err != nil {
		t.Fatal(err)
	}
	if err := st.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: "docA"}); err != nil {
		t.Fatal(err)
	}

	docs, err := st.ListTrackedDocs(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc after duplicate add, got %d", len(docs))
	}
}

func TestRemoveTrackedDoc(t *testing.T) {
	t.Run("removes the doc", func(t *testing.T) {
		st, pool := newStore(t)
		userID := insertUser(t, pool)
		ctx := context.Background()
		if err := st.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: "docA"}); err != nil {
			t.Fatal(err)
		}

		if err := st.RemoveTrackedDoc(ctx, userID, "docA"); err != nil {
			t.Fatal(err)
		}

		docs, err := st.ListTrackedDocs(ctx, userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 0 {
			t.Fatalf("expected 0 docs after remove, got %d", len(docs))
		}
	})

	t.Run("missing doc returns sentinel", func(t *testing.T) {
		st, pool := newStore(t)
		userID := insertUser(t, pool)

		err := st.RemoveTrackedDoc(context.Background(), userID, "no-such-doc")
		if !errors.Is(err, store.ErrTrackedDocNotFound) {
			t.Errorf("expected ErrTrackedDocNotFound, got %v", err)
		}
	})
}

func TestEnsureTabsOnConflict_DoesNotUnhideAPreviouslyHiddenTab(t *testing.T) {
	ctx := context.Background()
	st, pool := newStore(t)
	userID, tdID := seedTrackedDoc(t, st, pool, "docA")

	if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
		t.Fatal(err)
	}
	if err := st.HideTab(ctx, userID, "docA", "t1"); err != nil {
		t.Fatal(err)
	}
	if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
		t.Fatal(err)
	}

	rows, err := st.ListTabs(ctx, tdID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.TabID == "t1" && r.Visible {
			t.Error("EnsureTabs must not un-hide a hidden tab")
		}
	}
}

func TestHideShowTab(t *testing.T) {
	ctx := context.Background()

	t.Run("hide then show restores visibility", func(t *testing.T) {
		st, pool := newStore(t)
		userID, tdID := seedTrackedDoc(t, st, pool, "docA")
		if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}

		if err := st.HideTab(ctx, userID, "docA", "t1"); err != nil {
			t.Fatal(err)
		}
		rows, err := st.ListTabs(ctx, tdID)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r.TabID == "t1" && r.Visible {
				t.Error("tab t1 should be hidden after HideTab")
			}
		}

		if err := st.ShowTab(ctx, userID, "docA", "t1"); err != nil {
			t.Fatal(err)
		}
		rows, err = st.ListTabs(ctx, tdID)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r.TabID == "t1" && !r.Visible {
				t.Error("tab t1 should be visible after ShowTab")
			}
		}
	})

	t.Run("ownership: user B cannot hide or show user A's tab", func(t *testing.T) {
		st, pool := newStore(t)
		_, tdID := seedTrackedDoc(t, st, pool, "docA")
		userB := insertUser(t, pool)
		if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}

		if err := st.HideTab(ctx, userB, "docA", "t1"); !errors.Is(err, store.ErrTabNotFound) {
			t.Errorf("HideTab: expected ErrTabNotFound, got %v", err)
		}
		if err := st.ShowTab(ctx, userB, "docA", "t1"); !errors.Is(err, store.ErrTabNotFound) {
			t.Errorf("ShowTab: expected ErrTabNotFound, got %v", err)
		}
	})
}
