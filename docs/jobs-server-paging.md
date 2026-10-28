# Jobs list: server-side paging, filtering and sorting

Issue: [#938](https://github.com/O-Marsters-1997/job-scraper/issues/938). Settled in a feedback session; amends
ADR 0005, 0022 and 0031.

## Problem

`GET /jobs/all` returns every scored job from the last 90 days with no limit. The browser then filters
(`jobFilters.ts`), sorts (`jobSort.ts`) and pages (`JobsDataTable`, 10 rows). The page is slow to load and the
endpoint breaks ADR 0031. Companies already page on the server (offset, limit, total, #706), but nothing shares
that pattern.

## Decisions

- **Offset plus total.** The jobs list and the companies list both use it. Page numbers live in the URL.
  The Relevant rank is a computed expression, which a keyset cursor handles badly.
- **One list endpoint.** `GET /jobs` takes `page`, `page_size`, `sort` and the filters, and returns
  `{items, total}`. `GET /jobs/all` is deleted. The old keyset cursor on `/jobs` goes with it; nothing in the
  frontend calls it.
- **Page size 25**, default and fixed in the UI. The server clamps `page_size` to 100.
- **Three sorts, all in SQL:** `relevant` (default), `newest`, `best`. An unknown value means `relevant`.
  Header-click sorting and the "Custom" state are removed.
- **Relevant in SQL.** `rank = score * 0.5^(age_days / 3)`, unscored prior 50, age from `posted_at` falling
  back to `first_discovered_at` (clamped to 0 for future timestamps). Ties: `first_discovered_at` DESC, then
  `id`. This is what ADR 0022 already says. The client code used `first_discovered_at` then `scraped_at`, so
  jobs with a `posted_at` reorder slightly. The client sort and `jobSort.ts` are deleted.
- **Filters in SQL:** company, scored, min suitability, graded, seen, poster, favourite, source, work
  arrangement, salary, and text search. Seen, grade and favourite are per-user joins the query already makes.
- **Text search** is `ILIKE` over title, company slug, location and source. No index until it measures slow.
- **Salary** is parsed at ingest into `salary_min` and `salary_max` (nullable, plus currency and period if the
  parser yields them), backfilled from `salary_raw`. `salary_raw` stays the source of truth. The frontend
  `parseSalary` is ported to Go and deleted.
- **Selection** is a set of job IDs held in table state. It survives page changes and clears when a filter
  changes. Bulk actions send those IDs. There is no "select all matching".
- **After a mutation** (grade, mark seen, exclude company) the row is patched in place, then the list
  refetches. A row that no longer matches drops out on the refetch, and the page clamps to the last page.
- **Shared hook only.** `useServerTable` is extracted from `useCompanyPages`. It maps route search params to a
  request (page, sort, filters), resets page to 1 on a filter change, clamps to the last page when `total`
  shrinks, and uses `keepPreviousData`. Backend DTOs stay per endpoint. Page, sort and filters for both lists
  live in the URL.

## Out of scope

- Select all matching a filter.
- Sortable column headers.
- A `pg_trgm` or full-text index for search.
- A shared backend `Page[T]` type.

## Performance

`ORDER BY suitability_score` has no supporting index today. `posted_at`, `first_discovered_at`, `source`,
`work_arrangement` and company kind are not indexed either. The backend ticket checks `EXPLAIN` on realistic
data and adds only the indexes the plans need. The set is one user's open, scored jobs from 90 days, so a
scan after the `job_scores` join may be enough.

## Tickets

1. Parsed salary columns: migration, sqlc, Go parser, ingest, backfill.
2. Paged, filtered, sorted `GET /jobs`; delete `/jobs/all`; indexes.
3. `useServerTable`, the jobs table on it, companies migrated, ID-persistent selection, optimistic patch,
   demo mock.

Ticket 2 depends on 1 for the salary filter. Ticket 3 depends on 2.
