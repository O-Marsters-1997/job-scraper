# Plan: Cron-scraping + long-running worker (single binary)

## Context

Currently `cmd/main.go` runs the scraper once and then exits; the worker runs in a goroutine but the process terminates after `scraper.Run` returns. We want a single binary that (1) runs the worker forever, dequeuing from Valkey and processing jobs, and (2) runs scrapers on a schedule using in-process cron, with one schedule defined for WIS only for now.

---

## Goals

- **Long-running process**: Main goroutine blocks on the worker loop so the process stays up and keeps dequeuing.
- **Scheduled scraping**: Use `github.com/robfig/cron/v3` to run the scraper for WIS on a configurable cron expression (default e.g. every 6 hours).
- **Single binary**: No separate cron daemon or second process; one deployment runs both worker and cron.
- **Graceful shutdown**: On SIGINT/SIGTERM, cron stops, worker exits, defers run, process exits.

---

## Changes

### 1. Add dependency

```bash
go get github.com/robfig/cron/v3
```

### 2. `cmd/main.go` — refactor lifecycle

**Current**: Start worker in goroutine, call `scraper.Run` on main, process exits after scraper.

**New**:

1. **Setup** (unchanged): Logger, signal context, DB, Valkey queue, all with defers.
2. **Cron goroutine**: Create a cron scheduler; add one job for WIS with a schedule (see step 3). Start the cron. In the job func, call `scraper.Run(ctx, []sources.Source{wis.New()}, db, q)` (or a small helper that builds the source list for "wis"). Use the same `ctx` so shutdown cancels in-flight scrapes. Optionally: run scraper once on startup before starting cron (so first run isn’t delayed by the first tick).
3. **Main blocks on worker**: Call `worker.Run(ctx, q, handler, pollInterval)` **on the main goroutine** (no `go`). When `ctx` is cancelled, `worker.Run` returns and `main` exits; defers run (cron is stopped via `ctx` if the job checks `ctx` before/during scrape, and you can add `defer c.Stop()` for cleanliness).
4. **Shutdown**: `signal.NotifyContext` already provides `ctx`; pass it to both cron job and worker. When signal is received, `cancel()` runs, worker’s select gets `ctx.Done()` and returns, main exits. Optionally call `c.Stop()` in a defer so the cron scheduler stops cleanly.

**Concrete structure**:

- `ctx, cancel := signal.NotifyContext(...)`
- `db := ...`, `q := ...` (unchanged)
- `c := cron.New(cron.WithSeconds()` or standard 5-field — decide if you need seconds)
- Schedule for WIS: single expression (constant or env, e.g. `"0 0 */6 * * *"` for every 6 hours if using 6-field, or `"0 */6 * * *"` for 5-field).
- `c.AddFunc(schedule, func() { ... run scraper for WIS ... })` — inside the func, check `ctx.Done()` or pass `ctx` into `scraper.Run`.
- `c.Start()` then `defer c.Stop()`
- (Optional) run scraper once for WIS here so first crawl isn’t delayed.
- `worker.Run(ctx, q, handler, 5*time.Second)` on main (no goroutine).

### 3. Schedule for WIS only

- **Default schedule**: Choose one and document it (e.g. every 6 hours).
  - 5-field (robfig/cron): `"0 */6 * * *"` = at minute 0 of every 6th hour.
  - 6-field (optional): `"0 0 */6 * * *"` = at 0s, 0min of every 6th hour.
- **Configurable (optional for this plan)**: Env var e.g. `WIS_CRON="0 */6 * * *"`; if unset, use the default. Only WIS is configured; no need for a generic “schedules for all sources” map yet.

### 4. Cron job implementation detail

- The func registered with `c.AddFunc` should:
  - Call `scraper.Run(ctx, []sources.Source{wis.New()}, db, q)`.
  - Do not block cron’s runner for long; `scraper.Run` is synchronous, so the cron job will run to completion each time. If you need to avoid overlapping runs, you can add a mutex or “skip if previous run still in progress” later (out of scope for this plan).
- Pass the same `ctx` into `scraper.Run` so that when the process is shutting down, an in-progress scrape can be cancelled.

### 5. Worker remains the “main” process

- `worker.Run` stays as-is: loop with ticker, `q.Dequeue`, handler. No changes to `internal/worker/worker.go` unless you want to add a small improvement (e.g. log when waiting). The important change is only in `main`: run worker on the main goroutine so the process lifetime is tied to the worker.

---

## Files to touch

| File              | Change                                                                 |
|-------------------|------------------------------------------------------------------------|
| `go.mod` / `go.sum` | Add `github.com/robfig/cron/v3`                                      |
| `cmd/main.go`     | Add cron scheduler; one WIS job with default (or env) schedule; run worker on main; graceful stop of cron on shutdown |

---

## Verification

- Build and run the binary; worker should log periodically (e.g. every 5s) when no job is dequeued, or process jobs when present.
- Trigger a scrape (wait for first cron tick or run one scrape on startup) and confirm jobs are enqueued; worker should dequeue and run the handler.
- Send SIGINT; process should exit cleanly (cron stopped, worker returned, no panic).

---

## Out of scope (for later)

- Different cron expressions per source in config (only WIS is defined here).
- Locking to prevent overlapping scrape runs.
- HTTP server or health checks.
