# job-scraper

An automated pipeline that crawls startup job boards, deduplicates listings, enriches them with AI scoring, and emails a filtered digest.

## Overview

The entry point is `cmd/worker` — a single long-running process that runs three things concurrently:

1. **Orchestrator** — a cron-driven loop that iterates each configured source on a schedule (default `0 */6 * * *`) and either saves jobs directly or enqueues URLs for detail fetching, depending on the source type.
2. **Worker loop** — dequeues URLs one at a time, fetches the detail page via the owning source's `GetDetails`, then ingests the result.
3. **Notification cron** — a daily digest (configurable via `NOTIFY_DIGEST_CRON`), plus optional per-ingest alerts.

### Two source types

The central design concept is `Source.NeedsDetail()`:

- **ATS sources** (`NeedsDetail() == false`): greenhouse, lever, ashby, workable, recruitee, personio. Hit a JSON API and return fully-populated `dto.Job`s in one call. Jobs are ingested directly — no queue round-trip.
- **HTML sources** (`NeedsDetail() == true`): wis (Work In Startups, always on), linkedin, indeed. Scrape an HTML listing page, yielding URLs plus partial card metadata. Each URL is enqueued; the worker fetches the detail page separately.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│ Orchestrator (cron, per source)                                  │
│                                                                  │
│  ATS source (NeedsDetail=false)                                  │
│    Iterate() ──► relevance gate ──► Ingest ──► DB + score        │
│    (JSON API, full job)                                          │
│                                                                  │
│  HTML source (NeedsDetail=true)                                  │
│    Iterate() ──► rewrite aggregator URLs ──► filter new          │
│    (listing page, URL + card)   (detect.RewriteToATS)  (NewURLs) │
│              ──► relevance gate ──► Queue (Valkey sorted set)    │
└──────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌──────────────────────────────────────────────────────────────────┐
│ Worker (HTML sources only)                                       │
│                                                                  │
│  Queue.Dequeue() ──► sources.Dispatch() ──► Ingest ──► DB        │
│                      (GetDetails by URL)                         │
└──────────────────────────────────────────────────────────────────┘
                                │
                                ▼
                    Ingest: validate → Save → ScoreAndSave → NotifyNewJob
```

### Component roles

| Package | Role |
|---|---|
| `cmd/worker` | Entry point; registers sources, wires scorer/notifier, runs orchestrator + worker + digest cron |
| `internal/scraper` | Cron orchestrator; schedules scrapes, branches on `NeedsDetail`, enqueues or ingests |
| `internal/sources` | `Source` interface, `PaginatedBase` helper, `Dispatch` URL router |
| `internal/sources/wis` | HTML source — Work In Startups (always registered) |
| `internal/sources/greenhouse` | ATS source — Greenhouse JSON API |
| `internal/sources/lever` | ATS source — Lever JSON API |
| `internal/sources/ashby` | ATS source — Ashby JSON API |
| `internal/sources/workable` | ATS source — Workable JSON API |
| `internal/sources/recruitee` | ATS source — Recruitee JSON API |
| `internal/sources/personio` | ATS source — Personio XML API |
| `internal/sources/linkedin` | HTML source — LinkedIn (env-gated, needs residential proxy) |
| `internal/sources/indeed` | HTML source — Indeed (env-gated, needs residential proxy) |
| `internal/detect` | Classifies URLs by ATS type; `RewriteToATS` strips aggregator wrappers |
| `internal/queue` | Valkey-backed sorted-set queue; `ZADD NX` dedup, exponential backoff retry, dead-letter after 3 attempts |
| `internal/worker` | Sequential consumer loop; random 10–15s between items, 30s when empty |
| `internal/ingest` | validate → `db.Save` → `Scorer.ScoreAndSave` → `Notifier.NotifyNewJob` |
| `internal/score` | Heuristic relevance scorer (cheap, runs at scrape time as gate) + Claude suitability scorer (LLM, runs at ingest) |
| `internal/notify` | Resend-backed email; per-job alerts and daily digest, gated by suitability threshold |
| `internal/data/db` | pgx pool + sqlc-generated queries |
| `internal/dto` | Shared data transfer objects (`Job`, `QueuedJob`, `SearchConfig`) |
| `internal/proxy` | Per-source proxy tiers: direct / datacenter / residential |
| `cmd/snapshot` | CLI for capturing and rebasing HTML parser snapshots |

### Key design decisions

**`NeedsDetail()` splits the pipeline** — ATS sources collapse discovery + extraction into one cheap API call and skip the queue entirely. HTML sources use the two-phase URL-then-detail flow. The orchestrator branches on this at callback time.

**Two scoring stages** — Relevance (`HeuristicScorer`) is keyword-only, runs cheap at scrape time as a gate before enqueueing or saving. Suitability (`ClaudeScorer`) is an LLM call on the full job description, runs once at ingest, and gates notifications. See ADRs 0005 and 0006.

**Valkey sorted set as queue** — `jobs:pending` sorted by Unix-ms timestamp. `ZADD NX` deduplicates at enqueue; `ZPOPMIN` gives atomic dequeue. Retry uses exponential backoff (`30s × attempt`); after 3 attempts URLs move to `jobs:deadletter`. See ADR 0008.

**Aggregator URL rewriting** — When an HTML source (LinkedIn/Indeed) yields a URL that wraps an underlying ATS URL, `detect.RewriteToATS` extracts the real URL before enqueueing. This means the worker sees a clean ATS URL and routes it to the correct source. See ADR 0004.

**Per-source proxy tiering** — Direct for ATS APIs, datacenter for friendly HTML boards (wis), residential for hostile aggregators (LinkedIn, Indeed). Configured via `ProxyTier` in `sources.Config`. See ADR 0009.

**sqlc for queries** — All SQL is hand-written in `internal/data/sqlc/queries/` and compiled to type-safe Go. No ORM.

**Snapshot testing for parsers** — HTML snapshots are committed alongside each HTML source. Tests parse the snapshot and compare against a committed JSON fixture. See [Adding a new source](docs/sources/adding-a-source.md).

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

All sources except `wis` are opt-in via environment variables:

| Source | Env var | Value |
|---|---|---|
| Greenhouse | `GREENHOUSE_BOARDS` | Comma-separated board tokens, e.g. `acmecorp,widgets-inc` |
| Lever | `LEVER_BOARDS` | Comma-separated board tokens |
| Ashby | `ASHBY_BOARDS` | Comma-separated board tokens |
| Workable | `WORKABLE_BOARDS` | Comma-separated board tokens |
| Recruitee | `RECRUITEE_BOARDS` | Comma-separated board tokens |
| Personio | `PERSONIO_BOARDS` | Comma-separated board tokens |
| LinkedIn | `LINKEDIN_ENABLED` | `true` |
| Indeed | `INDEED_ENABLED` | `true` |

Scoring and notifications are also opt-in:

| Feature | Env var(s) |
|---|---|
| Relevance gate + suitability scoring | `SCORING_USER_ID` + `ANTHROPIC_API_KEY` |
| Email notifications | `RESEND_API_KEY` + `NOTIFY_EMAIL_TO` |
| Per-ingest email | `NOTIFY_ON_INGEST=true` |
| Daily digest | `NOTIFY_DIGEST_ENABLED=true` (default) + `NOTIFY_DIGEST_CRON` |
| Proxy (datacenter) | `PROXY_DATACENTER_URL` |
| Proxy (residential) | `PROXY_RESIDENTIAL_URL` |

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

./bin/worker   # long-running process
./bin/api      # API server (separate binary)
```

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
