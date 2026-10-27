package store_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/notify/store"
)

func TestStore_InsertNotification(t *testing.T) {
	t.Parallel()
	pool := pgtest.New(t)
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Go Engineer", "fp")

	id, err := store.New(pool).InsertNotification(t.Context(), userID, jobID, "title", "body")
	if err != nil {
		t.Fatalf("InsertNotification() = %v", err)
	}
	var unread bool
	err = pool.QueryRow(t.Context(),
		`SELECT read_at IS NULL FROM notifications WHERE id = $1 AND user_id = $2 AND job_id = $3`, id, userID, jobID).Scan(&unread)
	if err != nil || !unread {
		t.Errorf("row lookup = unread %v, err %v; want one unread row", unread, err)
	}
}

func TestStore_Notifications(t *testing.T) {
	t.Parallel()
	pool := pgtest.New(t)
	st := store.New(pool)
	jobID := pgtest.InsertJob(t, pool, "Go Engineer", "fp")
	ctx := t.Context()

	insert := func(user, title string) string {
		t.Helper()
		id, err := st.InsertNotification(ctx, user, jobID, title, "body")
		if err != nil {
			t.Fatalf("InsertNotification() = %v", err)
		}
		return id
	}
	titles := func(user string) []string {
		t.Helper()
		got, err := st.ListNotifications(ctx, user)
		if err != nil {
			t.Fatalf("ListNotifications() = %v", err)
		}
		var out []string
		for _, n := range got {
			out = append(out, n.Title)
		}
		return out
	}
	readAt := func(id string) *time.Time {
		t.Helper()
		var at *time.Time
		if err := pool.QueryRow(ctx, `SELECT read_at FROM notifications WHERE id = $1`, id).Scan(&at); err != nil {
			t.Fatalf("read_at lookup: %v", err)
		}
		return at
	}

	t.Run("lists unread before read, newest first, scoped to the user", func(t *testing.T) {
		t.Parallel()
		alice, bob := pgtest.InsertUser(t, pool), pgtest.InsertUser(t, pool)
		old := insert(alice, "old")
		insert(alice, "new")
		insert(alice, "newest")
		insert(bob, "bobs")
		if err := st.MarkNotificationRead(ctx, alice, old); err != nil {
			t.Fatalf("MarkNotificationRead() = %v", err)
		}
		if diff := cmp.Diff([]string{"newest", "new", "old"}, titles(alice)); diff != "" {
			t.Errorf("alice titles (-want +got):\n%s", diff)
		}
	})

	t.Run("caps the list at 50", func(t *testing.T) {
		t.Parallel()
		carol := pgtest.InsertUser(t, pool)
		for range 55 {
			insert(carol, "n")
		}
		if got := len(titles(carol)); got != 50 {
			t.Errorf("len(ListNotifications()) = %d, want 50", got)
		}
	})

	t.Run("another user cannot mark read", func(t *testing.T) {
		t.Parallel()
		alice, bob := pgtest.InsertUser(t, pool), pgtest.InsertUser(t, pool)
		id := insert(bob, "b")
		if err := st.MarkNotificationRead(ctx, alice, id); err != nil {
			t.Fatalf("MarkNotificationRead() = %v", err)
		}
		if err := st.MarkAllNotificationsRead(ctx, alice); err != nil {
			t.Fatalf("MarkAllNotificationsRead() = %v", err)
		}
		if readAt(id) != nil {
			t.Error("read_at set by another user, want unread")
		}
	})

	t.Run("re-reading keeps the original read_at", func(t *testing.T) {
		t.Parallel()
		dave := pgtest.InsertUser(t, pool)
		id := insert(dave, "d")
		if err := st.MarkNotificationRead(ctx, dave, id); err != nil {
			t.Fatalf("MarkNotificationRead() = %v", err)
		}
		first := readAt(id)
		if first == nil {
			t.Fatal("read_at = nil after read")
		}
		if err := st.MarkNotificationRead(ctx, dave, id); err != nil {
			t.Fatalf("MarkNotificationRead() = %v", err)
		}
		if err := st.MarkAllNotificationsRead(ctx, dave); err != nil {
			t.Fatalf("MarkAllNotificationsRead() = %v", err)
		}
		if got := readAt(id); !got.Equal(*first) {
			t.Errorf("read_at = %v, want %v", got, first)
		}
	})

	t.Run("read-all marks only the user's rows", func(t *testing.T) {
		t.Parallel()
		erin := pgtest.InsertUser(t, pool)
		frank := pgtest.InsertUser(t, pool)
		mine, theirs := insert(erin, "e"), insert(frank, "f")
		if err := st.MarkAllNotificationsRead(ctx, erin); err != nil {
			t.Fatalf("MarkAllNotificationsRead() = %v", err)
		}
		if readAt(mine) == nil || readAt(theirs) != nil {
			t.Errorf("read_at mine=%v theirs=%v, want set and nil", readAt(mine), readAt(theirs))
		}
	})
}
