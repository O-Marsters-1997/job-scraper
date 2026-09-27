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
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

var userSeq atomic.Int64

func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	username := fmt.Sprintf("user-%s-%d", t.Name(), userSeq.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`,
		username).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func TestAddTrackedDoc(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)

	if err := st.AddTrackedDoc(context.Background(), dto.AddTrackedDocInput{UserID: userID, DocID: "docA"}); err != nil {
		t.Fatal(err)
	}

	docs, err := st.ListTrackedDocs(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].DocID != "docA" {
		t.Fatalf("docs = %+v, want one doc with DocID=docA", docs)
	}
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
}

func TestRemoveTrackedDoc_MissingReturnsSentinel(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)

	err := st.RemoveTrackedDoc(context.Background(), userID, "no-such-doc")
	if !errors.Is(err, store.ErrTrackedDocNotFound) {
		t.Errorf("expected ErrTrackedDocNotFound, got %v", err)
	}
}

func seedDoc(t *testing.T, st *store.Store, pool *pgxpool.Pool, docID string) (userID, trackedDocID string) {
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

func TestEnsureTabs(t *testing.T) {
	ctx := context.Background()

	t.Run("inserts default-visible rows", func(t *testing.T) {
		st, pool := newStore(t)
		_, tdID := seedDoc(t, st, pool, "docA")

		if err := st.EnsureTabs(ctx, tdID, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
			t.Fatal(err)
		}
		rows, err := st.ListTabs(ctx, tdID)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		for _, r := range rows {
			if !r.Visible {
				t.Errorf("tab %q should default to visible", r.TabID)
			}
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		st, pool := newStore(t)
		_, tdID := seedDoc(t, st, pool, "docA")

		if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.EnsureTabs(ctx, tdID, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
			t.Fatal(err)
		}
		rows, err := st.ListTabs(ctx, tdID)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
	})

	t.Run("does not un-hide a previously hidden tab", func(t *testing.T) {
		st, pool := newStore(t)
		userID, tdID := seedDoc(t, st, pool, "docA")

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
	})

	t.Run("no-op for empty tabIDs", func(t *testing.T) {
		st, pool := newStore(t)
		_, tdID := seedDoc(t, st, pool, "docA")

		if err := st.EnsureTabs(ctx, tdID, nil, nil); err != nil {
			t.Fatal(err)
		}
		rows, err := st.ListTabs(ctx, tdID)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 0 {
			t.Fatalf("expected 0 rows, got %d", len(rows))
		}
	})
}

func TestHideTab(t *testing.T) {
	ctx := context.Background()

	t.Run("sets visible to false", func(t *testing.T) {
		st, pool := newStore(t)
		userID, tdID := seedDoc(t, st, pool, "docA")

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
	})

	t.Run("returns sentinel for missing tab", func(t *testing.T) {
		st, pool := newStore(t)
		userID, tdID := seedDoc(t, st, pool, "docA")
		if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}

		err := st.HideTab(ctx, userID, "docA", "t-missing")
		if !errors.Is(err, store.ErrTabNotFound) {
			t.Errorf("expected ErrTabNotFound, got %v", err)
		}
	})

	t.Run("ownership: user B cannot hide user A's tab", func(t *testing.T) {
		st, pool := newStore(t)
		_, tdID := seedDoc(t, st, pool, "docA")
		userB := insertUser(t, pool)

		if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}
		err := st.HideTab(ctx, userB, "docA", "t1")
		if !errors.Is(err, store.ErrTabNotFound) {
			t.Errorf("expected ErrTabNotFound, got %v", err)
		}
	})
}

func TestShowTab(t *testing.T) {
	ctx := context.Background()

	t.Run("restores a hidden tab", func(t *testing.T) {
		st, pool := newStore(t)
		userID, tdID := seedDoc(t, st, pool, "docA")

		if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.HideTab(ctx, userID, "docA", "t1"); err != nil {
			t.Fatal(err)
		}
		if err := st.ShowTab(ctx, userID, "docA", "t1"); err != nil {
			t.Fatal(err)
		}
		rows, err := st.ListTabs(ctx, tdID)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r.TabID == "t1" && !r.Visible {
				t.Error("tab t1 should be visible after ShowTab")
			}
		}
	})

	t.Run("returns sentinel for missing tab", func(t *testing.T) {
		st, pool := newStore(t)
		userID, tdID := seedDoc(t, st, pool, "docA")
		if err := st.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
			t.Fatal(err)
		}

		err := st.ShowTab(ctx, userID, "docA", "t-missing")
		if !errors.Is(err, store.ErrTabNotFound) {
			t.Errorf("expected ErrTabNotFound, got %v", err)
		}
	})
}
