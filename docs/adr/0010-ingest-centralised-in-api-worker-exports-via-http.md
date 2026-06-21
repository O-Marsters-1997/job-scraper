# ADR 0010 — Ingest centralised in API; worker exports via HTTP

**Status:** Accepted

## Context

Before this change the worker binary (`cmd/worker`) owned the full ingest pipeline: after fetching a job it called `ingest.Ingester` directly, which persisted the job to Postgres, ran Claude suitability scoring, and sent notifications. This meant the worker needed database credentials, an Anthropic API key, and a Resend API key — and all ingest logic was duplicated between the worker and any future ingestion path.

There were two concrete problems:

1. **Coupling** — scraping and ingest concerns lived in the same binary, making it harder to evolve either independently. Adding a second ingestion path (e.g. a manual UI upload) would mean reimplementing or re-importing the same ingest logic.
2. **Blast radius of scoring/notification bugs** — a misconfigured scorer or notifier could break the scraper process, or vice versa.

The API server (`cmd/api`) already ran as a separate binary and owned the database. Adding a `POST /ingest` endpoint there gives a single ingest seam regardless of how jobs arrive.

## Decision

Move persist → score → notify into `cmd/api` behind a `POST /ingest` endpoint, protected by a shared bearer token (`INGEST_SERVICE_TOKEN`). The worker calls this endpoint via `scraper.APIExporter`, never touching the database directly.

The `JobExporter` interface (`scraper.BulkExport` / `Export`) is the egress port both orchestrator paths terminate at:

- **ATS path** (orchestrator): `Iterate → relevance gate → exporter.BulkExport → POST /ingest`
- **HTML path** (worker): `Dispatch → exporter.Export → POST /ingest`

Both paths share the same `APIExporter` instance wired in `cmd/worker/main.go`.

`ingest.Ingester` (validate → Save → ScoreAndSave → NotifyNewJob) is wired into `cmd/api/main.go`, not the worker.

## Consequences

- **Worker simplified** — `cmd/worker` no longer imports `internal/ingest`, `internal/score` (Claude), or `internal/notify`. It only needs DB credentials for the queue last-scraped timestamp and the relevance gate's search config.
- **Single ingest path** — all ingested jobs flow through `POST /ingest` regardless of origin. Scoring and notification logic lives in one place.
- **New env vars required** — `API_BASE_URL` (worker must know the API address) and `INGEST_SERVICE_TOKEN` (shared secret; set on both binaries).
- **Retry on export** — `APIExporter` retries 5xx responses up to 2 times with exponential backoff before failing. Network partitions between worker and API may surface as dropped jobs if all retries exhaust; treat as acceptable for a personal low-volume tool.
- **`POST /ingest` is dark-shipped** — guarded by `ServiceTokenMiddleware` (constant-time bearer token comparison); the route is not exposed to browser clients and does not appear in user-facing auth flows.
