# job-scraper

An automated pipeline that crawls startup job boards, deduplicates listings, and persists enriched job records to a database.

## Overview

The scraper operates in two decoupled phases:

1. **Crawl** — A cron-driven orchestrator iterates each configured source on a schedule, filters out already-known URLs, and enqueues new ones.
2. **Enrich** — A worker dequeues URLs one at a time, dispatches each to the owning source to fetch full job details, and upserts the result to the database.

Decoupling via a queue means crawl throughput and enrichment throughput are independently controlled, and the two phases can be rate-limited differently (crawl runs on a 6-hour cron; enrichment adds a random 10-15s delay between requests to be polite to target sites).

## Architecture

```
┌─────────────────────────────────────────────────────┐
│ Orchestrator (cron, per source)                     │
│                                                     │
│  Source.Iterate() ──► filter new URLs ──► Queue     │
│  (paginated listing)   (db.NewURLs)     (Valkey     │
│                                          sorted set) │
└─────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────┐
│ Worker                                              │
│                                                     │
│  Queue.Dequeue() ──► Source.GetDetails() ──► DB     │
│                      (detail page parse)   (upsert) │
└─────────────────────────────────────────────────────┘
```

### Component Roles

| Package | Role | Key Types |
|---|---|---|
| `internal/scraper` | Cron orchestrator; schedules scrapes, deduplicates, enqueues | `Orchestrator` |
| `internal/sources` | `Source` interface + `PaginatedBase` helper; URL routing via `Dispatch()` | `Source`, `PaginatedBase` |
| `internal/sources/wis` | Concrete scraper for [WorkInStartups.com](https://workinstartups.com) | `Scraper` |
| `internal/queue` | Valkey-backed sorted-set queue with `JobQueue` interface | `Queue`, `FakeQueue` |
| `internal/worker` | Sequential consumer loop; respects context cancellation | `Run()`, `HandlerFunc` |
| `internal/data/db` | pgx connection pool + sqlc-generated queries | `DB`, `Save`, `NewURLs`, `List` |
| `internal/dto` | Shared job data transfer object | `Job` |
| `cmd/snapshot` | CLI for capturing and rebasing HTML parser snapshots | `download`, `rebase` |

### Key Design Decisions

**Interface-first** — `Source`, `JobQueue`, and `JobProvider` are all interfaces. This keeps the orchestrator, worker, and database fully decoupled and trivially testable with fakes.

**sqlc for queries** — All SQL is written by hand in `internal/data/sqlc/queries/jobs.sql` and compiled to type-safe Go by sqlc. No ORM, no string interpolation, no runtime surprises.

**Valkey sorted set as queue** — URLs are stored with a Unix-millisecond score. `ZADD NX` gives free deduplication at enqueue time; `ZPOPMIN` gives atomic dequeue. The same Valkey instance tracks last-scrape timestamps per source.

**Snapshot testing for parsers** — HTML snapshots are committed alongside each source. Tests parse the snapshot HTML and compare against a committed JSON fixture. To update: `just cli rebase <source>`.

## Getting Started

**Prerequisites**: Docker, [just](https://github.com/casey/just), Go 1.23+

```sh
# Start PostgreSQL and Valkey
just up

# Apply migrations
just migrate-up

# Run the scraper
just run
```

Copy `.env.example` to `.env` and adjust if your local setup differs from the defaults.

## Adding a New Source

1. **Implement the `Source` interface** (`internal/sources/source.go`):
   ```go
   type Source interface {
       Cfg() Config
       CanHandle(url string) bool
       Iterate(ctx context.Context, fn func(ctx context.Context, urls []string) (stop bool, err error)) error
       GetDetails(ctx context.Context, url string) (dto.Job, error)
   }
   ```
   Embed `PaginatedBase` to get a pre-configured HTTP client, `CanHandle` implementation, and the `IteratePages` pagination helper.

2. **Register it in `cmd/main.go`**:
   ```go
   srcs := []sources.Source{wis.New(), mynewsource.New()}
   ```

3. **Add snapshot tests** — capture one list page and at least one detail page:
   ```sh
   # Capture a list-page snapshot
   just cli download <source> list_page1 <url>
   # Capture a detail-page snapshot (also writes a .url sidecar for rebase)
   just cli download <source> detail_job1 <url>
   # Generate JSON fixtures from current parser output
   just cli rebase <source>
   ```
   Then write a single test:
   ```go
   func TestSnapshots(t *testing.T) {
       sources.RunSnapshotTests(t, mynewsource.New())
   }
   ```
   The framework routes `list_*.html` to `ParseURLs` and `detail_*.html` to `ParseJobDetail` automatically. All fixtures use `[]dto.Job` as their JSON schema.

## Testing

```sh
just test        # unit + integration tests
just test-race   # with race detector
just ci          # fmt + lint + test
```

### Strategy

Tests are grouped by layer, each with a different scope and dependency profile:

| Layer | Approach | Dependencies |
|---|---|---|
| **Parsers** (`ParseURLs`, `ParseJobDetail`) | Snapshot tests — parse committed HTML fixtures, compare against committed JSON | None (zero network) |
| **DB** (`Save`, `NewURLs`, `List`) | Integration — real PostgreSQL via testcontainers | Docker |
| **Queue** (`Enqueue`, `Dequeue`, etc.) | Integration — real Valkey via testcontainers | Docker |
| **Orchestrator** | Unit — `MockQueue`, `MockJobProvider`, stub `Source` | None |
| **Worker** | Unit — `MockQueue`, stub `HandlerFunc` | None |

**Snapshot tests** are the primary pattern for new sources. All parser fixtures share a single `[]dto.Job` JSON schema — list pages produce URL-only entries, detail pages produce fully-populated entries. File prefix determines which parser the framework calls: `list_*.html` → `ParseURLs`, `detail_*.html` → `ParseJobDetail`.

**Integration tests** (DB and queue) prove the real infrastructure implementations work and are the only tests that require Docker. They are isolated — each test truncates or flushes to avoid cross-contamination.

**Unit tests** (orchestrator and worker) use mock implementations (`MockQueue` in `internal/queue/`, `MockJobProvider` in `internal/data/providers/`) that live outside `_test.go` so they can be imported across packages. These tests run in milliseconds with no infrastructure.

### Adding snapshot tests for a new source

```sh
# Capture fixtures
just cli download <source> list_page1 <listing-url>
just cli download <source> detail_job1 <detail-url>
# Generate JSON from current parser output
just cli rebase <source>
```

Write one test:
```go
func TestSnapshots(t *testing.T) {
    sources.RunSnapshotTests(t, mynewsource.New())
}
```

To update fixtures after a parser change: `just cli rebase <source>`.

## Future Directions

Natural extension points in the current architecture:

- **API / query layer** — `db.List()` already exists; a thin HTTP handler over it is the obvious next step for surfacing jobs to consumers.
- **Additional sources** — implement `Source`, register, done. The orchestrator and worker need no changes.
- **Notification hooks** — the worker's `HandlerFunc` is the right place to fan out to webhooks, email, or a notification queue after a successful upsert.
- **Observability** — the cron schedule and last-scrape timestamps are already tracked in Valkey; exposing these as metrics (Prometheus, etc.) is straightforward.
- **Filtering / relevance scoring** — a post-upsert enrichment stage could run keyword matching or an embedding model against `jobs.title` before notifying.
