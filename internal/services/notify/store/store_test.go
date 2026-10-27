package store_test

import (
	"testing"

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
