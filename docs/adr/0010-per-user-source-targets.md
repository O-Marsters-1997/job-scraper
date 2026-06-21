# ADR 0010 — Per-user Source Targets in Postgres

**Status:** Accepted

## Context

The worker previously read ATS board configuration from process-global environment variables (`GREENHOUSE_BOARDS`, `LEVER_BOARDS`, `ASHBY_BOARDS`, `WORKABLE_BOARDS`, `RECRUITEE_BOARDS`, `PERSONIO_BOARDS`, `LINKEDIN_ENABLED`, `INDEED_ENABLED`). This is wrong for a multi-user product: a single env-var toggle removes a source for all users, and there is no way for individual users to define which boards or URLs they care about.

ADR 0006 established that `search_config` and `job_scores` are per-user even in v1, with a single seeded user. It explicitly deferred per-user *fan-out* (scoring one job for every user) as a future behavioural change. This ADR extends that per-user pattern to **which sources get scraped**.

Sources are a fixed, **supported catalog** — a code-level registry (`internal/sources`) listing nine platforms: greenhouse, lever, ashby, workable, recruitee, personio, wis, linkedin, indeed. Users do not create new source types; they define which targets within supported sources they want watched.

## Decision

Introduce a `source_targets` table — per-user, per-source rows — to replace env-var board configuration.

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS source_targets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source     TEXT        NOT NULL,   -- must match a name in the source registry
    value      TEXT        NOT NULL,   -- board token (ATS) or URL (aggregator/HTML)
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, source, value)
);
```

The `value` field carries either a board token (for ATS sources like greenhouse) or a URL (for URL-based sources like linkedin). The distinction is derived from the source registry — no `kind` column is needed.

**Source registry** (`internal/sources/registry.go`) centralises the supported-source catalog, providing `SupportedSources()` and `LookupSource(name)` for validation. The `builder` package (`internal/sources/builder/build.go`) implements `BuildSources(targets []dto.SourceTarget) []sources.Source`, which groups board targets by source, always includes `wis`, and includes URL sources only when the user has a target for them.

**Worker cutover**: the worker loads `ListEnabledSourceTargets` (union across all users) from the DB at startup and passes the result to `builder.BuildSources`. The env-var blocks and `splitBoards` helper are removed.

**API**: source targets are managed via authenticated REST endpoints (`GET/POST/PATCH/DELETE /source-targets`), following the same handler/provider/store pattern as other per-user entities.

## Consequences

- `GREENHOUSE_BOARDS`, `LEVER_BOARDS`, `ASHBY_BOARDS`, `WORKABLE_BOARDS`, `RECRUITEE_BOARDS`, `PERSONIO_BOARDS`, `LINKEDIN_ENABLED`, and `INDEED_ENABLED` env vars are no longer read. Existing deployments must seed their boards via the API (or directly in the DB) before the new worker binary is deployed.
- The worker scrapes the **union** of all users' enabled targets into the shared `jobs` catalog. Scoring and notification remain pinned to a single `SCORING_USER_ID` (per ADR 0006, fan-out is still deferred).
- URL targets for linkedin, indeed, and wis are stored and validated now, but the Iterate paths for those sources are still stubs. Activating URL-driven discovery (feeding stored URLs into Iterate) is future work.
- Multi-user scraping is a runtime change only: seed more users' targets, and the union load naturally picks them up — no schema migration needed.
- `CONTEXT.md` relationships updated: "A User defines zero or more Source Targets; each Target maps to a supported Source."
