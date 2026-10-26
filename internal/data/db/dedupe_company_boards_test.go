package db_test

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

func TestDedupeCompanyBoardsMigration(t *testing.T) {
	t.Parallel()
	pool := pgtest.New(t)
	ctx := t.Context()

	raw, err := os.ReadFile("scripts/migrations/20261026135001_dedupe_company_boards_by_token_case.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, _ := strings.Cut(string(raw), "-- +goose Down")
	up = strings.TrimPrefix(up, "-- +goose Up")
	if _, err := pool.Exec(ctx, strings.TrimPrefix(down, "\n")); err != nil {
		t.Fatalf("down: %v", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	var companyID string
	if err := pool.QueryRow(ctx, `INSERT INTO companies (slug, name) VALUES ('experian', 'Experian') RETURNING id`).Scan(&companyID); err != nil {
		t.Fatal(err)
	}
	board := func(source, token, createdAt string) string {
		t.Helper()
		var id string
		err := pool.QueryRow(ctx,
			`INSERT INTO company_boards (company_id, source, board_token, status, verification_method, verified_at, created_at)
			 VALUES ($1, $2, $3, 'verified', 'test', NOW(), $4::timestamptz) RETURNING id`, companyID, source, token, createdAt).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	job := func(boardID string) string {
		t.Helper()
		id := pgtest.InsertJob(t, pool, "Role", "fp")
		exec(`UPDATE jobs SET primary_board_id = $1 WHERE id = $2`, boardID, id)
		return id
	}
	observe := func(boardID, jobID string, version int) {
		t.Helper()
		exec(`INSERT INTO board_job_observations (board_id, job_id, last_snapshot_version) VALUES ($1, $2, $3)`, boardID, jobID, version)
	}

	loser := board("smartrecruiters", "experian", "2026-01-01")
	survivor := board("smartrecruiters", "Experian", "2026-02-01")
	loserOnly, shared := job(loser), job(survivor)
	survivorExtra := job(survivor)
	observe(loser, loserOnly, 4)
	observe(loser, shared, 4)
	observe(survivor, shared, 7)
	caseSensitiveKept := board("greenhouse", "Foo", "2026-01-01")
	caseSensitiveRetired := board("greenhouse", "foo", "2026-02-01")
	other := board("ashby", "Other", "2026-01-01")

	exec(up)

	type row struct{ Token, Status string }
	boards := map[string]row{}
	rows, err := pool.Query(ctx, `SELECT id, board_token, status FROM company_boards`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		var r row
		if err := rows.Scan(&id, &r.Token, &r.Status); err != nil {
			t.Fatal(err)
		}
		boards[id] = r
	}
	rows.Close()
	if got := boards[survivor]; got != (row{"experian", "verified"}) {
		t.Errorf("survivor = %+v, want lowercased and verified", got)
	}
	if got := boards[loser]; got.Status != "retired" {
		t.Errorf("loser = %+v, want retired", got)
	}
	if got := boards[caseSensitiveKept]; got != (row{"Foo", "verified"}) {
		t.Errorf("greenhouse survivor = %+v, want token case kept", got)
	}
	if got := boards[caseSensitiveRetired]; got.Status != "retired" {
		t.Errorf("greenhouse loser = %+v, want retired", got)
	}
	if got := boards[other]; got != (row{"other", "verified"}) {
		t.Errorf("lone board = %+v, want lowercased and untouched otherwise", got)
	}

	for _, id := range []string{loserOnly, shared, survivorExtra} {
		var got string
		if err := pool.QueryRow(ctx, `SELECT primary_board_id::text FROM jobs WHERE id = $1`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != survivor {
			t.Errorf("job %s primary_board_id = %s, want survivor %s", id, got, survivor)
		}
	}

	obsRows, err := pool.Query(ctx, `SELECT board_id::text, job_id::text, last_snapshot_version FROM board_job_observations ORDER BY job_id`)
	if err != nil {
		t.Fatal(err)
	}
	type observation struct {
		Board, Job string
		Version    int
	}
	var gotObs, wantObs []observation
	for obsRows.Next() {
		var o observation
		if err := obsRows.Scan(&o.Board, &o.Job, &o.Version); err != nil {
			t.Fatal(err)
		}
		gotObs = append(gotObs, o)
	}
	obsRows.Close()
	wantObs = []observation{{survivor, loserOnly, 4}, {survivor, shared, 7}}
	if loserOnly > shared {
		wantObs[0], wantObs[1] = wantObs[1], wantObs[0]
	}
	if diff := cmp.Diff(wantObs, gotObs); diff != "" {
		t.Errorf("observations (-want +got):\n%s", diff)
	}

	var dupes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM (SELECT 1 FROM company_boards WHERE status <> 'retired'
		GROUP BY source, lower(board_token) HAVING count(*) > 1) d`).Scan(&dupes); err != nil || dupes != 0 {
		t.Errorf("duplicate groups = %d, %v, want 0", dupes, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO company_boards (company_id, source, board_token) VALUES ($1, 'smartrecruiters', 'EXPERIAN')`, companyID); err == nil {
		t.Error("inserting a case variant of an active Board succeeded, want unique violation")
	}
}
