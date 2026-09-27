package pgtest_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

func TestNew(t *testing.T) {
	t.Run("applies migrations", func(t *testing.T) {
		pool := pgtest.New(t)

		var version int64
		if err := pool.QueryRow(context.Background(), "SELECT max(version_id) FROM goose_db_version").Scan(&version); err != nil {
			t.Fatalf("query goose_db_version: %v", err)
		}
		if version == 0 {
			t.Errorf("version = 0, want migrations applied")
		}
	})

	t.Run("returns the same pool across calls", func(t *testing.T) {
		first := pgtest.New(t)
		second := pgtest.New(t)
		if first != second {
			t.Errorf("New returned different pools across calls, want the same shared pool")
		}
	})

	t.Run("truncates tables between calls", func(t *testing.T) {
		ctx := context.Background()
		pool := pgtest.New(t)

		if _, err := pool.Exec(ctx, "INSERT INTO companies (slug, name) VALUES ('pgtest-check', 'Pgtest Check')"); err != nil {
			t.Fatalf("insert: %v", err)
		}

		pool = pgtest.New(t)

		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM companies WHERE slug = 'pgtest-check'").Scan(&count); err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0 after truncation", count)
		}
	})
}
