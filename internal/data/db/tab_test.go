package db_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func truncateTrackedDocs(t *testing.T) {
	t.Helper()
	if _, err := testDB.Pool().Exec(
		context.Background(),
		"TRUNCATE tracked_docs, users CASCADE",
	); err != nil {
		t.Fatalf("truncate tracked docs: %v", err)
	}
}

func seedDoc(t *testing.T, ctx context.Context, username, docID string) (userID, trackedDocID string) {
	t.Helper()
	user, err := testDB.CreateUser(ctx, username, "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := testDB.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: user.ID, DocID: docID}); err != nil {
		t.Fatalf("AddTrackedDoc: %v", err)
	}
	docs, err := testDB.ListTrackedDocs(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListTrackedDocs: %v", err)
	}
	for _, d := range docs {
		if d.DocID == docID {
			return user.ID, d.ID
		}
	}
	t.Fatalf("seeded doc %q not found after insert", docID)
	return "", ""
}

func TestDB_EnsureTabs(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "inserts default-visible rows",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				_, tdID := seedDoc(t, ctx, "user1", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				rows, err := testDB.ListTabs(ctx, tdID)
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				want := []dto.Tab{
					{TrackedDocID: tdID, TabID: "t1", Title: "Tab 1", Visible: true},
					{TrackedDocID: tdID, TabID: "t2", Title: "Tab 2", Visible: true},
				}
				if diff := cmp.Diff(want, rows, cmpopts.IgnoreFields(dto.Tab{}, "ID", "CreatedAt")); diff != "" {
					t.Errorf("rows mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "is idempotent — does not duplicate rows",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				_, tdID := seedDoc(t, ctx, "user2", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1", "t2"}, []string{"Tab 1", "Tab 2"}); err != nil {
					t.Fatalf("first EnsureTabs: %v", err)
				}
				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1", "t2", "t3"}, []string{"Tab 1", "Tab 2", "Tab 3"}); err != nil {
					t.Fatalf("second EnsureTabs: %v", err)
				}
				rows, err := testDB.ListTabs(ctx, tdID)
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				if len(rows) != 3 {
					t.Fatalf("expected 3 rows, got %d", len(rows))
				}
			},
		},
		{
			name: "does not un-hide a previously hidden tab",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				userID, tdID := seedDoc(t, ctx, "user3", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				if err := testDB.HideTab(ctx, userID, "docA", "t1"); err != nil {
					t.Fatalf("HideTab: %v", err)
				}
				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("second EnsureTabs: %v", err)
				}
				rows, err := testDB.ListTabs(ctx, tdID)
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				for _, r := range rows {
					if r.TabID == "t1" && r.Visible {
						t.Error("EnsureTabs must not un-hide a hidden tab")
					}
				}
			},
		},
		{
			name: "no-op for empty tabIDs",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				_, tdID := seedDoc(t, ctx, "user4", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, nil, nil); err != nil {
					t.Fatalf("EnsureTabs(nil): %v", err)
				}
				rows, err := testDB.ListTabs(ctx, tdID)
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				if len(rows) != 0 {
					t.Fatalf("expected 0 rows, got %d", len(rows))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestDB_ListTabs(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "returns only rows for the given trackedDocID",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				// One user owning two docs — distinct trackedDocIDs, same username is fine.
				userID, tdA := seedDoc(t, ctx, "user5a", "docA")
				_, tdB := seedDoc(t, ctx, fmt.Sprintf("user5b-partner-%s", userID), "docB")

				if err := testDB.EnsureTabs(ctx, tdA, []string{"a1", "a2"}, []string{"Tab A1", "Tab A2"}); err != nil {
					t.Fatalf("EnsureTabs docA: %v", err)
				}
				if err := testDB.EnsureTabs(ctx, tdB, []string{"b1"}, []string{"Tab B1"}); err != nil {
					t.Fatalf("EnsureTabs docB: %v", err)
				}

				rows, err := testDB.ListTabs(ctx, tdA)
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				if len(rows) != 2 {
					t.Fatalf("expected 2 rows for docA, got %d", len(rows))
				}
				for _, r := range rows {
					if r.TrackedDocID != tdA {
						t.Errorf("row belongs to wrong doc: got %q, want %q", r.TrackedDocID, tdA)
					}
				}
			},
		},
		{
			name: "returns nil for unknown trackedDocID",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				rows, err := testDB.ListTabs(ctx, "00000000-0000-0000-0000-000000000000")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rows != nil {
					t.Errorf("expected nil rows, got %v", rows)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestDB_HideTab(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "sets visible to false",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				userID, tdID := seedDoc(t, ctx, "user6", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				if err := testDB.HideTab(ctx, userID, "docA", "t1"); err != nil {
					t.Fatalf("HideTab: %v", err)
				}
				rows, err := testDB.ListTabs(ctx, tdID)
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				for _, r := range rows {
					if r.TabID == "t1" && r.Visible {
						t.Error("tab t1 should be hidden after HideTab")
					}
				}
			},
		},
		{
			name: "returns sentinel for missing tab",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				userID, tdID := seedDoc(t, ctx, "user7", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				err := testDB.HideTab(ctx, userID, "docA", "t-missing")
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !isErr(err, providers.ErrTabNotFound) {
					t.Errorf("expected ErrTabNotFound, got %v", err)
				}
			},
		},
		{
			name: "ownership: user B cannot hide user A's tab",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				_, tdID := seedDoc(t, ctx, "userA", "docA")
				userB, err := testDB.CreateUser(ctx, "userB", "hash")
				if err != nil {
					t.Fatalf("CreateUser userB: %v", err)
				}

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				hideErr := testDB.HideTab(ctx, userB.ID, "docA", "t1")
				if hideErr == nil {
					t.Fatal("expected error when hiding another user's tab, got nil")
				}
				if !isErr(hideErr, providers.ErrTabNotFound) {
					t.Errorf("expected ErrTabNotFound, got %v", hideErr)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestDB_ShowTab(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "restores a hidden tab to visible",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				userID, tdID := seedDoc(t, ctx, "user10", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				if err := testDB.HideTab(ctx, userID, "docA", "t1"); err != nil {
					t.Fatalf("HideTab: %v", err)
				}
				if err := testDB.ShowTab(ctx, userID, "docA", "t1"); err != nil {
					t.Fatalf("ShowTab: %v", err)
				}
				rows, err := testDB.ListTabs(ctx, tdID)
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				for _, r := range rows {
					if r.TabID == "t1" && !r.Visible {
						t.Error("tab t1 should be visible after ShowTab")
					}
				}
			},
		},
		{
			name: "returns sentinel for missing tab",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				userID, tdID := seedDoc(t, ctx, "user11", "docA")

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				err := testDB.ShowTab(ctx, userID, "docA", "t-missing")
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !isErr(err, providers.ErrTabNotFound) {
					t.Errorf("expected ErrTabNotFound, got %v", err)
				}
			},
		},
		{
			name: "ownership: user B cannot restore user A's tab",
			run: func(t *testing.T) {
				truncateTrackedDocs(t)
				userA, tdID := seedDoc(t, ctx, "user12a", "docA")
				userB, err := testDB.CreateUser(ctx, "user12b", "hash")
				if err != nil {
					t.Fatalf("CreateUser userB: %v", err)
				}

				if err := testDB.EnsureTabs(ctx, tdID, []string{"t1"}, []string{"Tab 1"}); err != nil {
					t.Fatalf("EnsureTabs: %v", err)
				}
				if err := testDB.HideTab(ctx, userA, "docA", "t1"); err != nil {
					t.Fatalf("HideTab: %v", err)
				}
				showErr := testDB.ShowTab(ctx, userB.ID, "docA", "t1")
				if showErr == nil {
					t.Fatal("expected error when restoring another user's tab, got nil")
				}
				if !isErr(showErr, providers.ErrTabNotFound) {
					t.Errorf("expected ErrTabNotFound, got %v", showErr)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func isErr(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return err == target
}
