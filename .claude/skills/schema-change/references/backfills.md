# Backfills

Grounded in `scripts/migrations/20260923000004_create_company_boards.sql`,
which is a real end-to-end example: it creates a table, backfills it from
existing data, and quarantines the rows that don't fit.

## INSERT ... SELECT, in the same migration

```sql
INSERT INTO company_boards (company_id, source, board_token, status, verification_method, verified_at)
SELECT DISTINCT ON (c.ats_source, c.ats_token) c.id, c.ats_source, c.ats_token,
       'verified', 'legacy_import', NOW()
FROM companies c
WHERE c.ats_source IS NOT NULL AND c.ats_token IS NOT NULL
ORDER BY c.ats_source, c.ats_token, c.first_seen_at, c.id;
```

The backfill runs as a plain statement between the `CREATE TABLE` and
`-- +goose Down` — no separate migration, no application-code script. Goose
runs each migration in a transaction by default, so the schema change and
the data it depends on land or roll back together.

## DISTINCT ON for one row per group, deterministically

`company_boards` has a `UNIQUE (source, board_token)` constraint, but
multiple `companies` rows can share the same `(ats_source, ats_token)`
pair (that's the bug this migration is fixing). `DISTINCT ON (c.ats_source,
c.ats_token)` picks exactly one row per pair; the `ORDER BY` after it
(`c.first_seen_at, c.id`) decides *which* one — oldest company first, `id`
as a tiebreaker so re-running against the same data is deterministic.
Without the matching `ORDER BY`, `DISTINCT ON` picks an arbitrary row and
the migration isn't reproducible.

## A dead-letter table for what doesn't fit

```sql
CREATE TABLE board_backfill_issues (
    source TEXT NOT NULL,
    board_token TEXT NOT NULL,
    candidate_company_ids UUID[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source, board_token)
);

INSERT INTO board_backfill_issues (source, board_token, candidate_company_ids)
SELECT ats_source, ats_token, ARRAY_AGG(id ORDER BY first_seen_at, id)
FROM companies
WHERE ats_source IS NOT NULL AND ats_token IS NOT NULL
GROUP BY ats_source, ats_token
HAVING COUNT(*) > 1;
```

The `DISTINCT ON` above silently drops every company row except the one it
picked — that's fine for satisfying the `UNIQUE` constraint, but it would
lose the fact that a collision happened. `board_backfill_issues` records
every `(source, board_token)` pair that had more than one candidate company
(`HAVING COUNT(*) > 1`), with the full list of company ids that collided, so
the ambiguity is queryable later instead of silently discarded. Reach for
this pattern whenever a backfill has to make a lossy choice — one row wins,
others don't fit the new constraint — rather than letting the losers vanish.
