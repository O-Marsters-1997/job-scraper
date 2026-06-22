# ADR 0012 — Filter sources and scrape-now on-demand flow

**Status:** Accepted

## Context

ADR 0010 (per-user source targets) introduced a `source_targets` table with a `value` column whose meaning is determined by the source kind: a board token for ATS sources, a URL for URL-based sources. WIS was listed as a URL source, but a single WIS target was pre-baked into the builder and always active — it did not benefit from the per-user target model.

WIS searches are keyword-driven. A user wants to watch `"product engineer"` jobs, optionally filtered by region. Neither a static URL nor a board token cleanly represents this: the search URL is constructed at scrape time from the keyword and optional structured params. Storing a pre-constructed URL in `value` also means changes to the WIS query string format require re-seeding all targets.

A second unrelated problem: after a user adds a new source target (especially a filter source with a keyword), they must wait until the next scheduled scrape tick — up to 6 hours — to see results. There was no way to trigger an immediate scrape for a single new target without restarting the worker.

## Decision

### `kindFilter` — a third source kind

Introduce `kindFilter` alongside the existing `kindBoard` and `kindURL` in `internal/sources/registry.go`:

- `value` stores the keyword/search string (e.g. `"product engineer"`).
- Structured parameters (e.g. region) are stored in a new `filters JSONB` column on `source_targets`.
- Filter sources declare their accepted parameters via `[]FilterField` in the registry; `LookupFilterFields(name)` returns them.
- `IsFilterSource(name)` gates the builder and handler into the filter path.

WIS is re-classified as `kindFilter` with one optional `FilterField{Name: "region"}`.

`builder.BuildSources` accumulates `wis.Search{Keywords: t.Value, Region: t.Filters["region"]}` entries from all enabled WIS targets and constructs a single `wis.Scraper` with all searches. This replaces the hardcoded single-search WIS instance.

The `Create` handler validates submitted `filters` keys against the declared `FilterField` list and rejects unknown keys or missing required fields with HTTP 400.

### `scrape_now` — on-demand scrape via queue

When a user POSTs a new source target with `scrape_now: true`, the `Create` handler enqueues a `dto.ScrapeRequest{Target: t}` on the `scrape:requests` Valkey list (RPUSH). This is fire-and-forget from the HTTP path.

The worker runs a second polling loop, `worker.RunScrapeRequests`, in a separate goroutine alongside the existing detail-fetch `Run` loop. It LPOPs from `scrape:requests` and calls `orchestrator.ScrapeTarget(ctx, target)`.

`Orchestrator.ScrapeTarget` is a new method that:
1. Builds a one-off source via the `buildTarget` callback registered with `WithSourceReloader`.
2. Calls `run(ctx, src)` directly — skipping `runIfReady` and therefore the `MinScrapeInterval` gate.
3. Does NOT write `scrape:last`, so the regular scheduled rotation continues unaffected.

`WithSourceReloader(buildAll, buildOne)` is an optional opt-in; without it, `ScrapeTarget` returns an error and `buildAll` falls back to the static source slice.

Failures in `RunScrapeRequests` are best-effort and logged, but not retried. The regular cron tick will cover any missed scrape on the next interval.

## Consequences

- **`source_targets` schema change** — a `filters JSONB NOT NULL DEFAULT '{}'` column is added. Existing rows are unaffected (default empty object).
- **WIS is no longer always-on** — if a user has no enabled WIS targets, no WIS scraper is built. Deployments that previously relied on the always-on WIS scraper must seed at least one WIS target.
- **`kindFilter` sources skip the URL-prefix validation** — the `Create` handler branches on `IsFilterSource` before the existing URL-prefix check, so filter targets are validated against their `FilterField` declarations instead.
- **`CONTEXT.md` additions** — `Filter Source`, `FilterField`, `ScrapeRequest` terms added.
- **No retry on scrape-now failure** — by design; the next cron tick is the recovery path. Do not add Nack/retry logic to `RunScrapeRequests`.
- **`buildTarget` must build a minimal single-target source** — callers registering `WithSourceReloader` are responsible for ensuring `buildOne` returns a correctly configured source for the given target kind (board, URL, or filter). The orchestrator does not inspect the target kind itself.
