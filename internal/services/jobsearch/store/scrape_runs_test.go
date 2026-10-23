package store_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

type scrapeRunRow struct {
	Trigger, Stop string
	Finished      bool
	Pages         int
	Listings      int
	New           int
	Elsewhere     int
	Repeat        int
	Deferred      int
	Retries       int
}

func readScrapeRun(t *testing.T, pool *pgxpool.Pool, runID string) scrapeRunRow {
	t.Helper()
	var r scrapeRunRow
	err := pool.QueryRow(t.Context(), `SELECT trigger, COALESCE(stop_reason, ''), finished_at IS NOT NULL,
		pages, listings, new_listings, elsewhere, repeat_listings, deferred, retries
		FROM scrape_runs WHERE id = $1::uuid`, runID).
		Scan(&r.Trigger, &r.Stop, &r.Finished, &r.Pages, &r.Listings, &r.New, &r.Elsewhere, &r.Repeat, &r.Deferred, &r.Retries)
	if err != nil {
		t.Fatalf("read scrape run %s: %v", runID, err)
	}
	return r
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func startRun(t *testing.T, st *store.Store, id string) dto.SourceTarget {
	t.Helper()
	target, err := st.StartSourceTargetRun(t.Context(), id, nil)
	if err != nil {
		t.Fatalf("StartSourceTargetRun(%s) err = %v", id, err)
	}
	return target
}

func TestScrapeRunRecording(t *testing.T) {
	t.Parallel()

	t.Run("manual start, create-with-run and scheduled claim each record their trigger", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		ctx := t.Context()
		started := startRun(t, st, createTarget(t, st, userID, "linkedin", "manual").ID)
		created, err := st.CreateSourceTargetWithRun(ctx, userID, "linkedin", "created", true, nil, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		hourly := 60
		due := time.Now().Add(-time.Minute)
		scheduledTarget, err := st.CreateSourceTarget(ctx, userID, "linkedin", "scheduled", true, nil, dto.RunWindow{IntervalMinutes: &hourly}, &due)
		if err != nil {
			t.Fatal(err)
		}
		claimed, err := st.ClaimDueSourceTargets(ctx, 10, func(dto.RunWindow) *time.Time { return nil })
		if err != nil || len(claimed) != 1 || claimed[0].ID != scheduledTarget.ID {
			t.Fatalf("ClaimDueSourceTargets() = %v, %v, want only %s", claimed, err, scheduledTarget.ID)
		}

		for runID, want := range map[string]string{started.RunID: "manual", created.RunID: "manual", claimed[0].RunID: "scheduled"} {
			if got := readScrapeRun(t, pool, runID).Trigger; got != want {
				t.Errorf("trigger for run %s = %q, want %q", runID, got, want)
			}
		}
	})

	t.Run("cards are tallied as new, repeat or elsewhere", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		saveCards(t, st, createTarget(t, st, userID, "linkedin", "other"), dto.Job{URL: "https://example.com/j/elsewhere", Title: "A", CompanySlug: "acme"})
		target := startRun(t, st, createTarget(t, st, userID, "linkedin", "mine").ID)
		saveCards(t, st, target, dto.Job{URL: "https://example.com/j/repeat", Title: "B", CompanySlug: "acme"})

		saveCards(t, st, target,
			dto.Job{URL: "https://example.com/j/repeat", Title: "B", CompanySlug: "acme"},
			dto.Job{URL: "https://example.com/j/elsewhere", Title: "A", CompanySlug: "acme"},
			dto.Job{URL: "https://example.com/j/new", Title: "C", CompanySlug: "acme"},
		)

		got := readScrapeRun(t, pool, target.RunID)
		want := scrapeRunRow{Trigger: "manual", Listings: 4, New: 2, Elsewhere: 1, Repeat: 1}
		if got != want {
			t.Errorf("run = %+v, want %+v", got, want)
		}
	})

	t.Run("failed attempts, deferrals, pages and finish are recorded", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newUserStore(t)
		ctx := t.Context()
		target := startRun(t, st, createTarget(t, st, userID, "linkedin", "mine").ID)
		for attempt := 1; attempt <= 2; attempt++ {
			if err := st.ScrapeRunAttemptFailed(ctx, target.RunID, 0, attempt, "boom"); err != nil {
				t.Fatal(err)
			}
		}
		for _, call := range []func() error{
			func() error { return st.ScrapeRunDeferred(ctx, target.RunID) },
			func() error { return st.ScrapeRunPageDone(ctx, target.RunID) },
			func() error { return st.FinishScrapeRun(ctx, target.RunID, dto.StopPageCap) },
			func() error { return st.FinishScrapeRun(ctx, target.RunID, dto.StopFailed) },
		} {
			if err := call(); err != nil {
				t.Fatal(err)
			}
		}

		got := readScrapeRun(t, pool, target.RunID)
		want := scrapeRunRow{Trigger: "manual", Stop: "page_cap", Finished: true, Pages: 1, Deferred: 1, Retries: 2}
		if got != want {
			t.Errorf("run = %+v, want %+v", got, want)
		}
		if n := countRows(t, pool, "SELECT count(*) FROM scrape_run_errors WHERE run_id = $1::uuid", target.RunID); n != 2 {
			t.Errorf("error rows = %d, want 2", n)
		}
	})

	t.Run("an unknown run records nothing", func(t *testing.T) {
		t.Parallel()
		st, pool, _ := newUserStore(t)
		if err := st.ScrapeRunAttemptFailed(t.Context(), "00000000-0000-0000-0000-000000000001", 0, 1, "boom"); err != nil {
			t.Fatalf("ScrapeRunAttemptFailed() err = %v", err)
		}
		if n := countRows(t, pool, "SELECT count(*) FROM scrape_run_errors"); n != 0 {
			t.Errorf("error rows = %d, want 0", n)
		}
	})
}

func TestDeleteExpiredScrapeRuns(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	ctx := t.Context()
	old := startRun(t, st, createTarget(t, st, userID, "linkedin", "old").ID)
	recent := startRun(t, st, createTarget(t, st, userID, "linkedin", "recent").ID)
	if err := st.ScrapeRunAttemptFailed(ctx, old.RunID, 0, 1, "boom"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE scrape_runs SET started_at = NOW() - INTERVAL '31 days' WHERE id = $1::uuid", old.RunID); err != nil {
		t.Fatal(err)
	}

	if err := st.DeleteExpiredScrapeRuns(ctx); err != nil {
		t.Fatalf("DeleteExpiredScrapeRuns() err = %v", err)
	}

	if n := countRows(t, pool, "SELECT count(*) FROM scrape_runs WHERE id = $1::uuid", old.RunID); n != 0 {
		t.Errorf("old run rows = %d, want 0", n)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM scrape_run_errors"); n != 0 {
		t.Errorf("error rows = %d, want 0", n)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM scrape_runs WHERE id = $1::uuid", recent.RunID); n != 1 {
		t.Errorf("recent run rows = %d, want 1", n)
	}
}
