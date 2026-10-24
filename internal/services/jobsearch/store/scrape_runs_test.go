package store_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
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

func TestStore_ListScrapeRuns(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	ctx := t.Context()
	other := createTarget(t, st, pgtest.InsertUser(t, pool), "linkedin", "theirs")
	startRun(t, st, other.ID)

	mine := createTarget(t, st, userID, "linkedin", "mine")
	failing := createTarget(t, st, userID, "linkedin", "failing")
	var ids []string
	for _, id := range []string{mine.ID, mine.ID, mine.ID, failing.ID} {
		run := startRun(t, st, id)
		ids = append(ids, run.RunID)
		if _, err := pool.Exec(ctx, "UPDATE scrape_runs SET started_at = NOW() + $2 * INTERVAL '1 minute' WHERE id = $1::uuid", run.RunID, len(ids)); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.ScrapeRunAttemptFailed(ctx, ids[1], 2, 1, "boom"); err != nil {
		t.Fatal(err)
	}
	for id, stop := range map[string]dto.StopReason{ids[0]: dto.StopLastPage, ids[1]: dto.StopLastPage, ids[3]: dto.StopFailed} {
		if err := st.FinishScrapeRun(ctx, id, stop); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := pool.Exec(ctx, "UPDATE scrape_runs SET trigger = 'scheduled', source = 'greenhouse' WHERE id = $1::uuid", ids[2]); err != nil {
		t.Fatal(err)
	}

	t.Run("lists only the caller's runs newest first with outcome and errors", func(t *testing.T) {
		t.Parallel()
		runs, err := st.ListScrapeRuns(ctx, userID, dto.ScrapeRunListOptions{Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		type row struct {
			ID      string
			Outcome dto.ScrapeRunOutcome
			Errors  int
		}
		var got []row
		for _, r := range runs {
			got = append(got, row{r.ID, r.Outcome, len(r.Errors)})
		}
		want := []row{
			{ids[3], dto.OutcomeFailed, 0},
			{ids[2], dto.OutcomeRunning, 0},
			{ids[1], dto.OutcomeRetried, 1},
			{ids[0], dto.OutcomeSucceeded, 0},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("ListScrapeRuns (-want +got):\n%s", diff)
		}
		if runs[2].TargetValue != "mine" || runs[2].Errors[0].Message != "boom" {
			t.Errorf("run = %+v, want target mine with error boom", runs[2])
		}
	})

	t.Run("target narrows to one search", func(t *testing.T) {
		t.Parallel()
		runs, err := st.ListScrapeRuns(ctx, userID, dto.ScrapeRunListOptions{ScrapeRunFilter: dto.ScrapeRunFilter{TargetID: failing.ID}, Limit: 10})
		if err != nil || len(runs) != 1 || runs[0].ID != ids[3] {
			t.Errorf("ListScrapeRuns(target) = %v, %v, want only %s", runs, err, ids[3])
		}
	})

	t.Run("before pages without gaps or repeats", func(t *testing.T) {
		t.Parallel()
		first, err := st.ListScrapeRuns(ctx, userID, dto.ScrapeRunListOptions{Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		last := first[1]
		second, err := st.ListScrapeRuns(ctx, userID, dto.ScrapeRunListOptions{Limit: 2, BeforeStartedAt: last.StartedAt, BeforeID: last.ID})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, r := range append(first, second...) {
			got = append(got, r.ID)
		}
		want := []string{ids[3], ids[2], ids[1], ids[0]}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("paged ids (-want +got):\n%s", diff)
		}
	})

	t.Run("summary agrees with the list for any filter", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		cases := map[string]dto.ScrapeRunFilter{
			"none":                {},
			"outcomes widen":      {Outcome: []string{"failed", "retried"}},
			"facets narrow":       {Source: []string{"linkedin"}, Trigger: []string{"manual"}},
			"error message text":  {Q: "boom"},
			"search value text":   {Q: "FAIL", Outcome: []string{"running", "failed"}},
			"wildcard is literal": {Q: "%"},
			"one target":          {TargetID: failing.ID},
			"date range":          {From: now.Add(-time.Hour), To: now.Add(150 * time.Second)},
		}
		for name, filter := range cases {
			if filter.From.IsZero() {
				filter.From = now.Add(-time.Hour)
			}
			summary, err := st.ScrapeRunSummary(ctx, userID, dto.ScrapeRunSummaryOptions{
				ScrapeRunFilter: filter, Bucket: 5 * time.Minute, End: now.Add(time.Hour),
			})
			if err != nil {
				t.Fatalf("%s: ScrapeRunSummary() err = %v", name, err)
			}
			list := func(f dto.ScrapeRunFilter) []dto.ScrapeRun {
				runs, err := st.ListScrapeRuns(ctx, userID, dto.ScrapeRunListOptions{ScrapeRunFilter: f, Limit: 100})
				if err != nil {
					t.Fatalf("%s: ListScrapeRuns() err = %v", name, err)
				}
				return runs
			}
			if got := len(list(filter)); summary.Total != got {
				t.Errorf("%s: Total = %d, list has %d", name, summary.Total, got)
			}
			var bucketed int
			for _, b := range summary.Histogram {
				bucketed += b.OK + b.Retried + b.Failed
			}
			if bucketed != summary.Total {
				t.Errorf("%s: histogram counts %d runs, Total = %d", name, bucketed, summary.Total)
			}
			ignoring := map[dto.ScrapeRunFacet]func(*dto.ScrapeRunFilter){
				dto.FacetOutcome: func(f *dto.ScrapeRunFilter) { f.Outcome = nil },
				dto.FacetKind:    func(f *dto.ScrapeRunFilter) { f.Kind = nil },
				dto.FacetSource:  func(f *dto.ScrapeRunFilter) { f.Source = nil },
				dto.FacetTrigger: func(f *dto.ScrapeRunFilter) { f.Trigger = nil },
			}
			value := map[dto.ScrapeRunFacet]func(dto.ScrapeRun) string{
				dto.FacetOutcome: func(r dto.ScrapeRun) string { return string(r.Outcome) },
				dto.FacetKind:    func(r dto.ScrapeRun) string { return string(r.Kind) },
				dto.FacetSource:  func(r dto.ScrapeRun) string { return r.Source },
				dto.FacetTrigger: func(r dto.ScrapeRun) string { return string(r.Trigger) },
			}
			for facet, clear := range ignoring {
				widened := filter
				clear(&widened)
				want := map[string]int{}
				for _, r := range list(widened) {
					want[value[facet](r)]++
				}
				if diff := cmp.Diff(want, summary.Facets[facet]); diff != "" {
					t.Errorf("%s: %s facet (-want +got):\n%s", name, facet, diff)
				}
			}
		}
	})
}
