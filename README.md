# job-scraper

An automated pipeline that crawls startup job boards, deduplicates listings, enriches them with AI scoring, and emails a filtered digest.

## Overview

There are two long-running binaries:

- **`cmd/worker`** — runs the orchestrator, the queue consumer loop, and a session-cleanup cron. It scrapes sources and ships every discovered job to the API via `POST /ingest`.
- **`cmd/api`** — the HTTP API server. Handles all frontend routes, user auth, application tracking, CV templates, and the `POST /ingest` endpoint that persist/score/notifies incoming jobs.

`cmd/worker` runs three things concurrently:

1. **Orchestrator** — a cron-driven loop that iterates each configured source on a schedule (default `0 * * * *`, hourly) and either exports jobs directly (ATS sources) or enqueues URLs for detail fetching (HTML sources).
2. **Worker loop** — dequeues URLs one at a time, fetches the detail page via the owning source's `GetDetails`, then exports the result to the API.
3. **Session-cleanup cron** — a daily cron to purge expired sessions from the DB.

### Two source types

The central design concept is the optional `DetailFetcher` interface:

- **ATS sources** (implement `Source` only): greenhouse, lever, ashby, workable, recruitee, personio. Hit a JSON API and return fully-populated `dto.Job`s in one call. Jobs are exported directly via `APIExporter.BulkExport` — no queue round-trip.
- **HTML sources** (implement `Source` + `DetailFetcher`): wis (Work In Startups, always on), linkedin, indeed. Scrape an HTML listing page, yielding URLs plus partial card metadata. Each URL is enqueued; the worker fetches the detail page separately via `DetailFetcher.GetDetails`, then exports via `APIExporter.Export`.

The orchestrator branches at callback time: `if _, ok := src.(sources.DetailFetcher)` picks the HTML path; otherwise the ATS path runs.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│ Orchestrator (cron, per source)                    cmd/worker    │
│                                                                  │
│  ATS source (Source only, no DetailFetcher)                      │
│    Iterate() ──► relevance gate ──► APIExporter.BulkExport()     │
│    (JSON API, full job)                      │                   │
│                                              ▼                   │
│  HTML source (Source + DetailFetcher)    POST /ingest            │
│    Iterate() ──► rewrite aggregator URLs ──► filter new          │
│    (listing page, URL + card)  (detect.RewriteToATS)  (NewURLs)  │
│              ──► relevance gate ──► Queue (Valkey sorted set)    │
└──────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌──────────────────────────────────────────────────────────────────┐
│ Worker (HTML sources only)                         cmd/worker    │
│                                                                  │
│  Queue.Dequeue() ──► sources.Dispatch() ──► APIExporter.Export() │
│                      (GetDetails by URL)         │               │
│                                                  ▼               │
│                                            POST /ingest          │
└──────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│ POST /ingest  (service-token auth)                 cmd/api       │
│                                                                  │
│  validate → save → queue scoring effect                          │
│  scoring worker → per-user threshold → recipient email          │
└──────────────────────────────────────────────────────────────────┘
```

### Component roles

| Package | Role |
|---|---|
| `cmd/worker` | Loads source targets from DB, registers exporter and relevance gate, runs orchestrator + worker loop + session-cleanup cron |
| `cmd/api` | HTTP API server; frontend routes, user auth, application tracking, CV templates, ingest, and scoring outbox worker |
| `internal/scraper` | Cron orchestrator; schedules scrapes, branches on `DetailFetcher` interface, enqueues HTML URLs or bulk-exports ATS jobs; `JobExporter` egress interface + `APIExporter` HTTP implementation |
| `internal/sources` | `Source` and `DetailFetcher` interfaces, `PaginatedBase` helper, `Dispatch` URL router, supported-source registry |
| `internal/sources/builder` | Builds the active source set from per-user source targets loaded from the DB |
| `internal/sources/wis` | HTML source — Work In Startups (always active) |
| `internal/sources/greenhouse` | ATS source — Greenhouse JSON API |
| `internal/sources/lever` | ATS source — Lever JSON API |
| `internal/sources/ashby` | ATS source — Ashby JSON API |
| `internal/sources/workable` | ATS source — Workable JSON API |
| `internal/sources/recruitee` | ATS source — Recruitee JSON API |
| `internal/sources/personio` | ATS source — Personio XML API |
| `internal/sources/linkedin` | HTML source — LinkedIn (needs residential proxy) |
| `internal/sources/indeed` | HTML source — Indeed (needs residential proxy) |
| `internal/detect` | Classifies URLs by ATS type; `RewriteToATS` strips aggregator wrappers |
| `internal/queue` | Valkey-backed sorted-set queue; `ZADD NX` dedup, exponential backoff retry, dead-letter after 3 attempts |
| `internal/worker` | Sequential consumer loop; random 10–15s between items, 30s when empty |
| `internal/ingest` | Validates and saves jobs; scoring and notification delivery happen after the durable scoring effect |
| `internal/auth` | Cookie session middleware (user routes) + `ServiceTokenMiddleware` (bearer token for `POST /ingest`) |
| `internal/handlers` | HTTP handlers including `IngestHandler` for `POST /ingest` |
| `internal/score` | Heuristic relevance scorer and durable per-user Claude suitability worker |
| `internal/notify` | Resend-backed per-job email to each qualifying user's profile address |
| `internal/data/db` | pgx pool + sqlc-generated queries |
| `internal/dto` | Shared data transfer objects (`Job`, `QueuedJob`, `SearchConfig`) |
| `internal/proxy` | Per-source proxy tiers: direct / datacenter / residential |
| `cmd/snapshot` | CLI for capturing and rebasing HTML parser snapshots |

### Key design decisions

**`DetailFetcher` splits the pipeline** — ATS sources implement `Source` only; HTML sources also implement `DetailFetcher` (`CanHandle` + `GetDetails`). The orchestrator checks `src.(sources.DetailFetcher)` at callback time: HTML sources enqueue URLs for the worker; ATS sources export jobs immediately. This removes the need for a `NeedsDetail()` boolean on the interface — capability is declared by implementation, not by a flag.

**Ingest is centralised in the API** — both scraper paths (ATS bulk-export and HTML worker) terminate at `POST /ingest` via `APIExporter`. Persist, suitability scoring, and notification run in `cmd/api`, not in `cmd/worker`. The worker never touches the DB directly. This simplifies the worker binary and makes the ingest logic testable and deployable independently.

**Two scoring stages** — Relevance (`HeuristicScorer`) is keyword-only, runs cheap at scrape time as a gate before enqueueing or saving. Suitability (`ClaudeScorer`) is an LLM call on the full job description, runs once at ingest, and gates notifications. See ADRs 0005 and 0006.

**Valkey sorted set as queue** — `jobs:pending` sorted by Unix-ms timestamp. `ZADD NX` deduplicates at enqueue; `ZPOPMIN` gives atomic dequeue. Retry uses exponential backoff (`30s × attempt`); after 3 attempts URLs move to `jobs:deadletter`. See ADR 0008.

**Aggregator URL rewriting** — When an HTML source (LinkedIn/Indeed) yields a URL that wraps an underlying ATS URL, `detect.RewriteToATS` extracts the real URL before enqueueing. This means the worker sees a clean ATS URL and routes it to the correct source. See ADR 0004.

**Per-source proxy opt-in** — ATS APIs and cooperative HTML boards (wis) go direct; hostile aggregators (LinkedIn, Indeed) route through BrightData Web Unlocker (residential unblocking, CAPTCHA/JS handling). Configured via `UseProxy bool` in `sources.Config`. See ADR 0009.

**sqlc for queries** — All SQL is hand-written in `internal/data/sqlc/queries/` and compiled to type-safe Go. No ORM.

**Snapshot testing for parsers** — HTML snapshots are committed alongside each HTML source. Tests parse the snapshot and compare against a committed JSON fixture. See [Adding a new source](docs/sources/adding-a-source.md).

**Companies are a shared catalog; scoring is target-gated** — `companies` holds one row per company slug, auto-populated at ingest for any unseen `company_slug`, with an optional known ATS board (`ats_source`/`ats_token`; NULL means discovery-only). Tracking a company just enables its ATS `source_target` — no separate tracking table. Suitability scoring now only runs for users with a matching enabled target (exact company match for ATS jobs, any target on the source for discovery jobs), not for every credentialed user. See ADR 0016.

## Getting Started

**Prerequisites**: Docker, [just](https://github.com/casey/just), Go 1.26+

```sh
# Start PostgreSQL and Valkey
just up

# Apply migrations
just migrate-up

# Run the worker
just run
```

Copy `.env.example` to `.env` and adjust if your local setup differs from the defaults.

### Source registration

Sources are configured per-user via the REST API and stored in the database. `wis` is always active and requires no configuration. All other sources are opt-in.

**ATS sources** take a board token as the value:

| Source | `source` field | Example `value` |
|---|---|---|
| Greenhouse | `greenhouse` | `acmecorp` |
| Lever | `lever` | `my-startup` |
| Ashby | `ashby` | `widgetco` |
| Workable | `workable` | `techco` |
| Recruitee | `recruitee` | `startup-hq` |
| Personio | `personio` | `mycompany` |

**URL sources** take a search URL as the value:

| Source | `source` field | Example `value` |
|---|---|---|
| LinkedIn | `linkedin` | `https://www.linkedin.com/jobs/search/?keywords=engineer` |
| Indeed | `indeed` | `https://www.indeed.com/jobs?q=software+engineer` |

Manage source targets via the authenticated API:

```sh
GET    /source-targets          # list your configured sources
POST   /source-targets          # add a source: {"source":"greenhouse","value":"acmecorp"}
PATCH  /source-targets/{id}     # update: {"enabled"?: bool, "check_interval_minutes"?: int}
DELETE /source-targets/{id}     # remove a source
```

`check_interval_minutes` (default 360) has a 60-minute floor and controls how often the worker re-checks that specific target — see `ListDueSourceTargets` and ADR 0016. The worker loads due source targets each hourly tick and passes them to `sources/builder.BuildSources` to assemble the active source set.

Companies are a shared catalog auto-populated from ingested jobs; the API also lets you browse and track known ATS boards directly:

```sh
GET  /companies               # list companies, with your per-caller tracked/interval/job-count status
POST /companies                # resolve+add a company from a board URL: {"url":"...","track":true,"scrape_now":false}
PUT  /companies/{id}/tracking  # toggle tracking (enables/disables that company's source target): {"enabled":true}
```

`POST /companies` returns `422` if the URL doesn't resolve to a known ATS board.

Worker → API connection (required):

| Env var | Purpose |
|---|---|
| `API_BASE_URL` | Base URL of the running API server, e.g. `http://localhost:8080` (required by worker) |
| `INGEST_SERVICE_TOKEN` | Shared bearer token authenticating worker → `POST /ingest` (set on both worker and API) |

Scoring and notifications (configured on `cmd/api`):

| Feature | Env var(s) |
|---|---|
| Relevance gate (worker) | `SCORING_USER_ID` |
| Suitability scoring (API) | `SCORING_USER_ID` + `ANTHROPIC_API_KEY` |
| Per-user email notifications (API) | `RESEND_API_KEY` + `NOTIFY_EMAIL_FROM` (optional); recipient comes from `users.email` and threshold from each user's search config |
| BrightData Web Unlocker | `BRIGHTDATA_PROXY_URL` |

Existing operator accounts need `users.email` set (via Profile settings) to keep receiving notifications after this change. A null email silently skips delivery.

## VPS Deployment

These steps cover deploying to a fresh Linux VPS (e.g. Hetzner, DigitalOcean).

### 1. SSH access

```sh
ssh-keygen -t ed25519 -C "your@email.com"
ssh-copy-id root@<server-ip>
ssh root@<server-ip>
```

### 2. Install dependencies

```sh
# Go
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc

# just
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# goose (for migrations)
go install github.com/pressly/goose/v3/cmd/goose@latest
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc && source ~/.bashrc

# Docker (for Valkey)
curl -fsSL https://get.docker.com | sh
```

### 3. Clone and configure

```sh
git clone https://github.com/O-Marsters-1997/job-scraper.git
cd job-scraper
cp .env.example .env
```

Edit `.env` with your values. All values must be **quoted** — `just`'s dotenv parser requires this for values containing spaces or special characters:

```sh
NOTIFY_DIGEST_CRON="0 9 * * *"
```

**Database**: use any PostgreSQL instance — Docker locally, or a managed service like [Neon](https://neon.tech). Set `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_SSLMODE`.

> **Neon / pooler note**: if your `POSTGRES_HOST` contains `-pooler.` in the hostname, migrations will hang because `goose` uses session-level advisory locks incompatible with PgBouncer transaction mode. Use the direct (non-pooler) endpoint at port `5432` for `just migrate-up`. The pooler URL is fine at runtime.

**Valkey**:

```sh
docker run -d --name valkey -p 6379:6379 valkey/valkey:latest
```

Set `VALKEY_ADDR="localhost:6379"` in `.env`.

### 4. Run migrations and start

```sh
just migrate-up
just build

./bin/api      # API server — must start first (worker POSTs to it)
./bin/worker   # scraper + queue consumer
```

Both processes must run together. Set `API_BASE_URL` in `.env` to the address the API server listens on, and set matching `INGEST_SERVICE_TOKEN` on both. Suitability scoring and notifications are configured on the API server via `ANTHROPIC_API_KEY`, `RESEND_API_KEY`, etc.

To keep the process running after disconnect, use `systemd` or `screen`/`tmux`.

## Adding a New Source

See [docs/sources/adding-a-source.md](docs/sources/adding-a-source.md) for the full guide covering the ATS vs HTML decision, interface implementation, `PaginatedBase` helpers, snapshot testing, and registration.

## Testing

```sh
just test        # unit + integration tests
just test-race   # with race detector
just ci          # fmt + lint + generate + test + build
```

### Strategy

| Layer | Approach | Dependencies |
|---|---|---|
| **Parsers** (`ParseURLs`, `ParseJobDetail`) | Snapshot tests — parse committed HTML, compare against committed JSON | None |
| **DB** (`Save`, `NewURLs`, `List`) | Integration — real PostgreSQL via testcontainers | Docker |
| **Queue** (`Enqueue`, `Dequeue`, etc.) | Integration — real Valkey via testcontainers | Docker |
| **Orchestrator** | Unit — `FakeQueue`, stub `Source` | None |
| **Worker** | Unit — `FakeQueue`, stub handler | None |

**Snapshot tests** are the primary pattern for HTML sources. All fixtures use `[]dto.Job` as their JSON schema. File prefix determines which parser the framework calls: `list_*.html` → `ParseURLs`, `detail_*.html` → `ParseJobDetail`. The URL for detail parsing is read from the fixture's first entry.

**Integration tests** (DB and queue) use testcontainers and are the only tests that require Docker. Each test truncates or flushes to avoid cross-contamination.

**Unit tests** (orchestrator, worker, ingest, score) use fakes/mocks that live outside `_test.go` files so they can be imported across packages.

### Snapshot workflow

```sh
# Capture fixtures
just cli download <source> list_page1 <listing-url>
just cli download <source> detail_job1 <detail-url>

# Generate JSON from current parser output
just cli rebase <source>
```

Write one test per source:

```go
func TestSnapshots(t *testing.T) {
    sources.RunSnapshotTests(t, mysource.New())
}
```

To update fixtures after a parser change: `just cli rebase <source>`.
