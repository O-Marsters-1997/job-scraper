package db_test

import (
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/scripts/migrations"
)

func TestCheckSchemaNotAhead(t *testing.T) {
	t.Parallel()
	pool := pgtest.New(t)

	t.Run("bundled migrations match the database", func(t *testing.T) {
		t.Parallel()
		if err := db.CheckSchemaNotAhead(t.Context(), pool, migrations.FS); err != nil {
			t.Errorf("CheckSchemaNotAhead(current build) = %v, want nil", err)
		}
	})

	t.Run("database ahead of the build names the build version", func(t *testing.T) {
		t.Parallel()
		old := fstest.MapFS{"20260314000000_create_jobs.sql": {}}
		var dbVersion int64
		if err := pool.QueryRow(t.Context(), `SELECT max(version_id) FROM goose_db_version`).Scan(&dbVersion); err != nil {
			t.Fatal(err)
		}
		err := db.CheckSchemaNotAhead(t.Context(), pool, old)
		for _, want := range []string{"20260314000000", strconv.FormatInt(dbVersion, 10)} {
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("CheckSchemaNotAhead(old build) = %v, want error naming %s", err, want)
			}
		}
	})
}
