package db_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

func TestWisRegionToLocMigration(t *testing.T) {
	pool := pgtest.New(t)
	ctx := t.Context()
	userID := pgtest.InsertUser(t, pool)

	raw, err := os.ReadFile("scripts/migrations/20261002181840_wis_region_to_loc.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := strings.TrimPrefix(strings.SplitN(string(raw), "-- +goose Down", 2)[0], "-- +goose Up")

	rows := []struct{ source, value, filters string }{
		{"wis", "region-only", `{"region":"uk"}`},
		{"wis", "region-and-other", `{"region":"uk","remote":"1"}`},
		{"wis", "already-loc", `{"loc":"86383"}`},
		{"wis", "no-filters", `{}`},
		{"linkedin", "other-source", `{"region":"uk"}`},
		{"wis", "twin", `{"region":"uk"}`},
		{"wis", "twin", `{"loc":"86383"}`},
	}
	for _, r := range rows {
		_, err := pool.Exec(ctx,
			`INSERT INTO source_targets (user_id, source, value, filters) VALUES ($1, $2, $3, $4::jsonb)`,
			userID, r.source, r.value, r.filters)
		if err != nil {
			t.Fatalf("insert %s: %v", r.value, err)
		}
	}

	want := map[string]string{
		"region-only":      `{"loc": "86383"}`,
		"region-and-other": `{"loc": "86383", "remote": "1"}`,
		"already-loc":      `{"loc": "86383"}`,
		"no-filters":       `{}`,
		"other-source":     `{"region": "uk"}`,
		"twin":             `{"loc": "86383"}`,
	}

	for _, pass := range []string{"first run", "re-run"} {
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("%s: %v", pass, err)
		}
		for value, wantFilters := range want {
			var got string
			err := pool.QueryRow(ctx,
				`SELECT filters::text FROM source_targets WHERE value = $1 ORDER BY filters::text LIMIT 1`, value).Scan(&got)
			if err != nil {
				t.Fatalf("%s: read %s: %v", pass, value, err)
			}
			if got != wantFilters {
				t.Errorf("%s: %s filters = %s, want %s", pass, value, got, wantFilters)
			}
		}
	}
}
