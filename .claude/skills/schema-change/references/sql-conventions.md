# SQL conventions

Grounded in `scripts/migrations/20260923000004_create_company_boards.sql`
(the `company_boards` / `board_backfill_issues` migration) and cross-checked
against `internal/data/db/sqlc/schema.sql`.

## Primary keys and timestamps

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
...
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
```

Every table in this codebase uses a `UUID PRIMARY KEY DEFAULT
gen_random_uuid()` surrogate key (the two join tables that use composite
keys instead — `board_job_observations`, `candidate_discoveries` — are
exceptions where the key is genuinely two foreign keys, not an id). All
timestamp columns are `TIMESTAMPTZ`; a required one is `NOT NULL DEFAULT
NOW()`, an optional one (like `verified_at`, `retired_at` in the same
migration) is left nullable with no default.

## Enums as TEXT + CHECK

```sql
status TEXT NOT NULL DEFAULT 'candidate' CHECK (status IN ('candidate', 'verified', 'retired')),
```

There's no Postgres `ENUM` type anywhere in this schema. A fixed set of
string values is a `TEXT` column with a `CHECK (col IN (...))` constraint
and a sensible default. Same pattern elsewhere: `source_targets.run_status`
(`'idle', 'queued', 'running', 'succeeded', 'failed'`),
`job_candidates.detail_state` (`'unrequested', 'pending'`).

## Index naming

```sql
CREATE INDEX company_boards_company_status_idx ON company_boards (company_id, status);
```

`<table>_<cols>_idx`, all lowercase, underscore-joined. Holds across the
schema: `jobs_page_idx`, `jobs_open_page_idx`, `effect_outbox_user_job_idx`,
`board_poll_state_due_idx`.

## Foreign keys and cascade

```sql
company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
```

`ON DELETE CASCADE` is used when the child row has no meaning without the
parent — a `company_boards` row without its `company` is garbage, so it
cascades. Compare `applications.status_id UUID REFERENCES
application_statuses(id) ON DELETE SET NULL` in `schema.sql`: an
application still means something after its status category is deleted, so
that one nulls out instead of cascading. Pick based on whether the parent
*owns* the child row or merely *references* it.

## Cross-column invariants as table CHECK constraints

```sql
CHECK (status <> 'verified' OR (verification_method IS NOT NULL AND verified_at IS NOT NULL))
```

A rule that spans more than one column belongs in a table-level `CHECK`,
not application code — this one enforces that a `verified` board always
carries how and when it was verified. `tracked_companies.check_interval_minutes
INT NOT NULL DEFAULT 360 CHECK (check_interval_minutes >= 60)` (in
`schema.sql`) is the single-column version of the same idea.
