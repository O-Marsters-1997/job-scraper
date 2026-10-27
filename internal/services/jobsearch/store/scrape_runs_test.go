package store_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
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

func TestStore_ListSourceHealth(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	ctx := t.Context()
	startRun(t, st, createTarget(t, st, pgtest.InsertUser(t, pool), "linkedin", "theirs").ID)
	createTarget(t, st, userID, "linkedin", "idle")
	busy := createTarget(t, st, userID, "linkedin", "busy")

	for i := range 6 {
		run := startRun(t, st, busy.ID)
		if _, err := pool.Exec(ctx, "UPDATE scrape_runs SET started_at = NOW() - $2 * INTERVAL '1 minute', new_listings = 2 WHERE id = $1::uuid", run.RunID, 10-i); err != nil {
			t.Fatal(err)
		}
		stop := dto.StopLastPage
		if i == 5 {
			stop = dto.StopFailed
		}
		if i == 4 {
			if err := st.ScrapeRunAttemptFailed(ctx, run.RunID, 0, 1, "boom"); err != nil {
				t.Fatal(err)
			}
		}
		if err := st.FinishScrapeRun(ctx, run.RunID, stop); err != nil {
			t.Fatal(err)
		}
	}
	startRun(t, st, busy.ID)

	got, err := st.ListSourceHealth(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("ListSourceHealth() returned %d rows, want 2 (the caller's searches)", len(got))
	}
	byValue := map[string]dto.SourceHealth{}
	for _, h := range got {
		byValue[h.TargetValue] = h
	}
	if h := byValue["idle"]; len(h.Recent) != 0 || h.LastRunAt != nil {
		t.Errorf("idle = %+v, want no runs", h)
	}
	h := byValue["busy"]
	wantRecent := []dto.ScrapeRunOutcome{dto.OutcomeSucceeded, dto.OutcomeSucceeded, dto.OutcomeSucceeded, dto.OutcomeRetried, dto.OutcomeFailed}
	if diff := cmp.Diff(wantRecent, h.Recent); diff != "" {
		t.Errorf("busy Recent (-want +got):\n%s", diff)
	}
	if h.New != 10 || h.Retries != 1 || h.LastRunAt == nil {
		t.Errorf("busy = %+v, want New 10 over the last 5 finished runs, Retries 1 and a last run", h)
	}
}

func TestStore_ListRecentNewJobs(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	ctx := t.Context()
	mine := createTarget(t, st, userID, "linkedin", "mine")
	theirs := createTarget(t, st, pgtest.InsertUser(t, pool), "linkedin", "theirs")
	seed := func(target dto.SourceTarget, title string, age time.Duration) {
		t.Helper()
		run := startRun(t, st, target.ID)
		if _, err := pool.Exec(ctx, "UPDATE scrape_runs SET started_at = NOW() - $2::float8 * INTERVAL '1 second' WHERE id = $1::uuid", run.RunID, age.Seconds()); err != nil {
			t.Fatal(err)
		}
		_, err := pool.Exec(ctx, `WITH j AS (
			INSERT INTO jobs (title, location, url, company_slug, source) VALUES ($2, 'London', 'https://x.test/' || $2 || $3, 'acme', 'linkedin') RETURNING id)
			INSERT INTO scrape_run_job_changes (run_id, job_id, change, snapshot)
			SELECT $1::uuid, j.id, 'new', jsonb_build_object('Title', $2::text, 'Location', 'London', 'CompanySlug', 'acme') FROM j`, run.RunID, title, target.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	seed(mine, "recent", time.Hour)
	seed(mine, "old", 48*time.Hour)
	seed(theirs, "other user", time.Hour)

	got, err := st.ListRecentNewJobs(ctx, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	want := []dto.SourceNewJob{{TargetID: mine.ID, Job: dto.Job{Title: "recent", Location: "London", CompanySlug: "acme"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ListRecentNewJobs() (-want +got):\n%s", diff)
	}
}

func TestStore_BoardRuns(t *testing.T) {
	t.Parallel()
	env := verifiedBoardFixture(t)
	ctx := t.Context()
	other := pgtest.InsertUser(t, env.pool)
	trackCompany(t, env.st, other, env.companyID, 60)
	outsider := pgtest.InsertUser(t, env.pool)

	run := dto.BoardRunStart{ID: uuid.NewString(), BoardID: env.board.ID, Source: "greenhouse", Trigger: dto.TriggerManual, StartedAt: time.Now()}
	if err := env.st.BoardRunAttemptFailed(ctx, run, 1, "boom"); err != nil {
		t.Fatal(err)
	}
	jobs := make([]dto.Job, 4)
	for i := range jobs {
		jobs[i], _ = saveJob(t, env.st, env.job(i+1))
	}
	tally := dto.BoardTally{Claimed: true, Listed: 5, Elsewhere: 1, Changes: []dto.JobChange{
		{JobID: jobs[0].ID, Kind: dto.JobChangeNew},
		{JobID: jobs[1].ID, Kind: dto.JobChangeNew},
		{JobID: jobs[2].ID, Kind: dto.JobChangeRemoved},
		{JobID: jobs[3].ID, Kind: dto.JobChangeReopened},
	}}
	if err := env.st.FinishBoardRun(ctx, run, dto.StopLastPage, tally); err != nil {
		t.Fatal(err)
	}
	if err := env.st.FinishBoardRun(ctx, run, dto.StopFailed, dto.BoardTally{Changes: tally.Changes[:1]}); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, env.pool, "SELECT count(*) FROM scrape_run_job_changes WHERE run_id = $1 AND snapshot->>'Title' = 'Role 1' AND change = 'new'", run.ID); n != 1 {
		t.Errorf("new change rows with a snapshot = %d, want 1", n)
	}
	if n := countRows(t, env.pool, "SELECT count(*) FROM scrape_run_job_changes WHERE run_id = $1", run.ID); n != 4 {
		t.Errorf("change rows = %d, want 4 from the first finish only", n)
	}

	if n := countRows(t, env.pool, "SELECT count(*) FROM scrape_runs WHERE kind = 'board'"); n != 1 {
		t.Fatalf("board runs = %d, want 1 shared by every tracking user", n)
	}
	for _, user := range []string{env.userID, other} {
		runs, err := env.st.ListScrapeRuns(ctx, user, dto.ScrapeRunListOptions{Limit: 10})
		if err != nil || len(runs) != 1 {
			t.Fatalf("ListScrapeRuns(%s) = %d runs, %v, want 1", user, len(runs), err)
		}
		got := runs[0]
		if got.Kind != dto.ScrapeRunBoard || got.CompanyName != "Poll Co" || got.CompanyID != env.companyID || got.BoardID != env.board.ID {
			t.Errorf("run = %+v, want the Poll Co board run", got)
		}
		if got.Listings != 5 || got.New != 2 || got.Elsewhere != 1 || got.Removed != 1 || got.Reopened != 1 || got.Outcome != dto.OutcomeRetried || len(got.Errors) != 1 {
			t.Errorf("run = %+v, want the finished tally, retried, one error", got)
		}
		wantChanges := []dto.ScrapeRunJobChange{
			{JobID: jobs[0].ID, Kind: dto.JobChangeNew, Title: "Role 1"},
			{JobID: jobs[1].ID, Kind: dto.JobChangeNew, Title: "Role 2"},
			{JobID: jobs[2].ID, Kind: dto.JobChangeRemoved, Title: "Role 3"},
			{JobID: jobs[3].ID, Kind: dto.JobChangeReopened, Title: "Role 4"},
		}
		if diff := cmp.Diff(wantChanges, got.Changes); diff != "" {
			t.Errorf("Changes (-want +got):\n%s", diff)
		}
	}
	if runs, err := env.st.ListScrapeRuns(ctx, outsider, dto.ScrapeRunListOptions{Limit: 10}); err != nil || len(runs) != 0 {
		t.Errorf("ListScrapeRuns(outsider) = %d runs, %v, want none", len(runs), err)
	}
	if _, err := env.pool.Exec(ctx, "UPDATE jobs SET title = 'Renamed' WHERE id = $1", jobs[0].ID); err != nil {
		t.Fatal(err)
	}
	snaps, err := env.st.ListScrapeRunJobSnapshots(ctx, other, run.ID)
	if err != nil || len(snaps) != 4 {
		t.Fatalf("ListScrapeRunJobSnapshots() = %d snapshots, %v, want 4", len(snaps), err)
	}
	var first struct{ Title string }
	if err := json.Unmarshal(snaps[0].Snapshot, &first); err != nil {
		t.Fatal(err)
	}
	if snaps[0].JobID != jobs[0].ID || snaps[0].Kind != dto.JobChangeNew || first.Title != "Role 1" {
		t.Errorf("first snapshot = %+v title %q, want job 1 as of the poll, not the renamed Job", snaps[0], first.Title)
	}
	if _, err := env.st.ListScrapeRunJobSnapshots(ctx, outsider, run.ID); !errors.Is(err, data.ErrNotFound) {
		t.Errorf("ListScrapeRunJobSnapshots(outsider) err = %v, want ErrNotFound", err)
	}
	if _, err := env.st.ListScrapeRunJobSnapshots(ctx, other, "nope"); !errors.Is(err, store.ErrInvalidID) {
		t.Errorf("ListScrapeRunJobSnapshots(bad id) err = %v, want ErrInvalidID", err)
	}
	byName := dto.ScrapeRunListOptions{ScrapeRunFilter: dto.ScrapeRunFilter{Q: "poll"}, Limit: 10}
	if runs, err := env.st.ListScrapeRuns(ctx, env.userID, byName); err != nil || len(runs) != 1 {
		t.Errorf("ListScrapeRuns(q=poll) = %d runs, %v, want the Company name to match", len(runs), err)
	}

	summaryFor := func(user, company string) dto.ScrapeRunSummary {
		t.Helper()
		from := time.Now().Add(-time.Hour)
		summary, err := env.st.ScrapeRunSummary(ctx, user, dto.ScrapeRunSummaryOptions{
			ScrapeRunFilter: dto.ScrapeRunFilter{CompanyID: company, From: from},
			Bucket:          time.Hour, End: time.Now(),
		})
		if err != nil {
			t.Fatalf("ScrapeRunSummary(company=%s) err = %v", company, err)
		}
		return summary
	}
	if got := summaryFor(env.userID, env.companyID); got.Total != 1 || got.New != 2 || got.Removed != 1 || got.LastStartedAt == nil {
		t.Errorf("summary(company) = %+v, want 1 run with New 2, Removed 1 and a last start", got)
	}
	if got := summaryFor(outsider, env.companyID); got.Total != 0 || got.LastStartedAt != nil {
		t.Errorf("summary(outsider) = %+v, want nothing for an untracked Company", got)
	}
	if got := summaryFor(env.userID, uuid.NewString()); got.Total != 0 {
		t.Errorf("summary(other company) = %+v, want nothing", got)
	}
	byCompany := dto.ScrapeRunListOptions{ScrapeRunFilter: dto.ScrapeRunFilter{CompanyID: env.companyID}, Limit: 10}
	if runs, err := env.st.ListScrapeRuns(ctx, env.userID, byCompany); err != nil || len(runs) != 1 {
		t.Errorf("ListScrapeRuns(company) = %d runs, %v, want 1", len(runs), err)
	}
	if runs, err := env.st.ListScrapeRuns(ctx, outsider, byCompany); err != nil || len(runs) != 0 {
		t.Errorf("ListScrapeRuns(outsider, company) = %d runs, %v, want none", len(runs), err)
	}
	invalid := dto.ScrapeRunListOptions{ScrapeRunFilter: dto.ScrapeRunFilter{CompanyID: "nope"}, Limit: 10}
	if _, err := env.st.ListScrapeRuns(ctx, env.userID, invalid); !errors.Is(err, store.ErrInvalidID) {
		t.Errorf("ListScrapeRuns(bad company) err = %v, want ErrInvalidID", err)
	}

	health, err := env.st.ListSourceHealth(ctx, env.userID)
	if err != nil || len(health) != 1 {
		t.Fatalf("ListSourceHealth() = %+v, %v, want the one tracked board", health, err)
	}
	if h := health[0]; h.Kind != dto.ScrapeRunBoard || h.CompanyName != "Poll Co" || h.Removed != 1 || len(h.Recent) != 1 {
		t.Errorf("health = %+v, want the board named by Company with its removed count", h)
	}

	if _, err := env.pool.Exec(ctx, "UPDATE scrape_runs SET started_at = NOW() - INTERVAL '31 days' WHERE id = $1::uuid", run.ID); err != nil {
		t.Fatal(err)
	}
	if err := env.st.DeleteExpiredScrapeRuns(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, env.pool, "SELECT count(*) FROM scrape_run_job_changes"); n != 0 {
		t.Errorf("change rows after the run expired = %d, want 0", n)
	}
}
