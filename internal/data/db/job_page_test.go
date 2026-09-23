package db_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

func TestPageJobsKeepsPositionUnderInsert(t *testing.T) {
	truncate(t)
	ctx := t.Context()
	company := "10000000-0000-0000-0000-000000000001"
	if _, err := testDB.Pool().Exec(ctx, `INSERT INTO companies (id,slug,name) VALUES ($1,'page-jobs-test-acme','Acme') ON CONFLICT (id) DO NOTHING`, company); err != nil {
		t.Fatal(err)
	}
	insert := func(id string, day int, closed bool) {
		t.Helper()
		_, err := testDB.Pool().Exec(ctx, `INSERT INTO jobs (id,title,location,url,company_slug,source,updated_at,scraped_at,description,company_id,closed_at) VALUES ($1::uuid,'Role','','https://example.com/'||$1::text,'acme','test',$2::timestamptz,$2::timestamptz,'full description',$3,CASE WHEN $4 THEN $2::timestamptz ELSE NULL END)`, id, time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC), company, closed)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("20000000-0000-0000-0000-000000000001", 1, false)
	insert("20000000-0000-0000-0000-000000000002", 2, false)
	insert("20000000-0000-0000-0000-000000000003", 3, true)
	options := providers.JobPageOptions{Limit: 1, Availability: "open", CompanyID: company}
	first, err := testDB.Page(ctx, "30000000-0000-0000-0000-000000000001", options)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].ID != "20000000-0000-0000-0000-000000000002" || first.Items[0].Description != "" {
		t.Fatalf("first page: %+v", first)
	}
	detail, err := testDB.GetJob(ctx, first.Items[0].ID, "30000000-0000-0000-0000-000000000001")
	if err != nil || detail.Description != "full description" {
		t.Fatalf("detail: %+v, %v", detail, err)
	}
	insert("20000000-0000-0000-0000-000000000004", 4, false)
	options.CursorTime, options.CursorID = first.Items[0].ScrapedAt, first.Items[0].ID
	second, err := testDB.Page(ctx, "30000000-0000-0000-0000-000000000001", options)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "20000000-0000-0000-0000-000000000001" {
		t.Fatalf("second page: %+v", second)
	}
	_, err = testDB.Pool().Exec(ctx, `INSERT INTO jobs (id,title,url,company_slug,source,updated_at,scraped_at) VALUES ('20000000-0000-0000-0000-000000000005','Legacy','https://example.com/legacy','page-jobs-test-acme','test','2025-12-31','2025-12-31')`)
	if err != nil {
		t.Fatal(err)
	}
	options.CursorTime, options.CursorID = second.Items[0].ScrapedAt, second.Items[0].ID
	third, err := testDB.Page(ctx, "30000000-0000-0000-0000-000000000001", options)
	if err != nil || len(third.Items) != 1 || third.Items[0].Title != "Legacy" {
		t.Fatalf("legacy page: %+v, %v", third, err)
	}
	closed, err := testDB.Page(ctx, "30000000-0000-0000-0000-000000000001", providers.JobPageOptions{Limit: 10, Availability: "closed", CompanyID: company})
	if err != nil || len(closed.Items) != 1 || closed.Items[0].ID != "20000000-0000-0000-0000-000000000003" {
		t.Fatalf("closed page: %+v, %v", closed, err)
	}
}

func BenchmarkPageJobs100k(b *testing.B) {
	truncate(b)
	ctx := b.Context()
	_, err := testDB.Pool().Exec(ctx, `INSERT INTO jobs (title,url,company_slug,source,updated_at,scraped_at) SELECT 'Role', 'https://benchmark.example/' || n, 'benchmark', 'test', NOW(), NOW() - n * INTERVAL '1 second' FROM generate_series(1,100000) AS n`)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := testDB.Pool().Exec(ctx, `ANALYZE jobs`); err != nil {
		b.Fatal(err)
	}
	rows, err := testDB.Pool().Query(ctx, `EXPLAIN SELECT id FROM jobs WHERE closed_at IS NULL ORDER BY scraped_at DESC, id DESC LIMIT 51`)
	if err != nil {
		b.Fatal(err)
	}
	var plan string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			b.Fatal(err)
		}
		plan += line + "\n"
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		b.Fatal(err)
	}
	if !strings.Contains(plan, "Index") {
		b.Fatalf("unexpected plan: %s", plan)
	}
	b.Logf("100k open jobs query plan: %s", plan)
	options := providers.JobPageOptions{Limit: 51, Availability: "open"}
	b.ResetTimer()
	for b.Loop() {
		if _, err := testDB.Page(ctx, "30000000-0000-0000-0000-000000000001", options); err != nil {
			b.Fatal(err)
		}
	}
}
