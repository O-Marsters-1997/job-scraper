# Plan: URL-first discovery with DB deduplication

## Context

Currently `Iterate` returns full `[]dto.Job` (all details, all pages), dedup is in-memory, and everything gets enqueued regardless of whether it's already in the DB. The new architecture splits scraping into two phases: this plan covers phase 1 — URL discovery only. We don't need any job detail fields (title, company, location) at this stage — those come in the details phase. So `FetchJobs` is renamed to `FetchURLs` and the HTML parsing is simplified to extract only `href` links from job cards. `Iterate` calls `FetchURLs` per page, filters against the DB, stops early when a page is fully known, and enqueues only new URLs.

---

## Changes

### 1. `internal/sources/source.go`

- Remove `FetchJobs` from the interface — replace with `FetchURLs`
- Add `URLFilter` type
- `Iterate` accepts the filter and returns `[]string`

```go
// URLFilter filters a batch of URLs down to those not yet known.
// Returning an empty slice signals the caller to stop iterating.
type URLFilter func(ctx context.Context, urls []string) ([]string, error)

type Source interface {
    Name() string
    FetchURLs(ctx context.Context) ([]string, error)
    Iterate(ctx context.Context, filter URLFilter) ([]string, error)
}
```

### 2. `internal/sources/wis/wis.go`

**Rename `ParseHTML` → `ParseURLs`** and simplify: instead of building a `dto.Job` per card, just extract the `href` from the `<a>` inside `<h2>`. No company, location, or title parsing needed.

```go
func ParseURLs(r io.Reader) ([]string, error) {
    // walk DOM, find divs with data-aid, extract href from h2 > a
}
```

**Rename `FetchJobs` → `FetchURLs`**: calls `ParseURLs` instead of `ParseHTML`, returns `[]string`.

**`fetchPage`**: returns `([]string, int, error)` instead of `([]dto.Job, int, error)`. Calls `ParseURLs` (and `ParseTotalCount` on page 1 as before).

**`Iterate`**: signature becomes `(ctx, filter URLFilter) ([]string, error)`.
Per page:

1. `pageURLs, total, err := s.fetchPage(ctx, p)`
2. `newURLs, err := filter(ctx, pageURLs)` — log err and continue on failure
3. `all = append(all, newURLs...)`
4. **Early stop**: if `len(newURLs) == 0`, break

**Remove** `dto` import — no longer needed in this file.

**Snapshot tests**: `ParseHTML` is gone; `wis_test.go` switches to `ParseURLs`. Snapshot JSON files become `[]string` (URL arrays) — rebase snapshots after the change.

### 3. `internal/sources/greenhouse/greenhouse.go`

- Rename `FetchJobs` → `FetchURLs`: extracts only `absolute_url` from the API response, returns `[]string`
- `Iterate(ctx, filter URLFilter) ([]string, error)`: calls `FetchURLs`, passes result through `filter`, returns new URLs

### 4. `internal/data/sqlc/queries/jobs.sql`

Add:

```sql
-- name: ExistingURLs :many
SELECT url FROM jobs WHERE url = ANY($1::text[]);
```

### 5. Run `sqlc generate`

Regenerates `internal/data/db/pgsqlc/jobs.sql.go`.

### 6. `internal/data/db/job.go`

Add:

```go
func (db *DB) FilterNewURLs(ctx context.Context, urls []string) ([]string, error) {
    existing, err := pgsqlc.New(db.pool).ExistingURLs(ctx, urls)
    // build set from existing, return urls not in set
}
```

### 7. `internal/data/providers/job.go`

Add `FilterNewURLs` to the interface:

```go
type JobProvider interface {
    UpsertJobs(ctx context.Context, jobs []dto.Job) error
    UpsertJob(ctx context.Context, job dto.Job) error
    FilterNewURLs(ctx context.Context, urls []string) ([]string, error)
}
```

### 8. `internal/scraper/scrape.go`

- `Run` accepts `providers.JobProvider`
- Constructs `filter sources.URLFilter` from `db.FilterNewURLs`
- Calls `src.Iterate(ctx, filter)` — receives `[]string` directly
- In-memory dedup across sources still applies
- Enqueues as before

```go
func Run(ctx context.Context, srcs []sources.Source, db providers.JobProvider, q *queue.Queue) error
```

### 9. `cmd/main.go`

- Add Postgres connection via env var `DATABASE_URL`
- Initialise `db, err := jobsdb.New(ctx, connStr)` + `defer db.Close()`
- Pass `db` to `scraper.Run`

---

## Critical files

| File                                        | Change                                                                                                                                 |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/sources/source.go`                | `FetchURLs` replaces `FetchJobs`; add `URLFilter`; update `Iterate`                                                                    |
| `internal/sources/wis/wis.go`               | `ParseURLs` replaces `ParseHTML`; `FetchURLs` replaces `FetchJobs`; `fetchPage` returns `[]string`; `Iterate` uses filter + early stop |
| `internal/sources/wis/wis_test.go`          | Switch to `ParseURLs`; rebase snapshots to `[]string` JSON                                                                             |
| `internal/sources/greenhouse/greenhouse.go` | `FetchURLs` replaces `FetchJobs`; update `Iterate`                                                                                     |
| `internal/data/sqlc/queries/jobs.sql`       | Add `ExistingURLs`                                                                                                                     |
| `internal/data/db/pgsqlc/jobs.sql.go`       | Regenerated                                                                                                                            |
| `internal/data/db/job.go`                   | Add `FilterNewURLs`                                                                                                                    |
| `internal/data/providers/job.go`            | Add `FilterNewURLs` to interface                                                                                                       |
| `internal/scraper/scrape.go`                | `Run` accepts `providers.JobProvider`, passes filter to `Iterate`                                                                      |
| `cmd/main.go`                               | DB init + pass to `Run`                                                                                                                |

## Verification

```bash
sqlc generate
go run ./cmd/snapshot rebase wis   # regenerate snapshot JSON as []string
go build ./...
go test ./...
```
