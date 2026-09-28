# Altering an existing table

Grounded in `scripts/migrations/20260706210858_add_jobs_closed_at.sql`, the
smallest real example of a table alteration in this repo:

```sql
-- +goose Up
ALTER TABLE jobs ADD COLUMN closed_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE jobs DROP COLUMN closed_at;
```

Two things worth noting even in a two-line migration:

## Nullable column, no default, no backfill

`closed_at` is added nullable with no `DEFAULT` and no backfill step. This
works because `NULL` *is* the correct value for every existing row — an
already-scraped job simply isn't known to be closed. This is the cheap
path: an `ALTER TABLE ADD COLUMN` with no default and no backfill is a
metadata-only change in Postgres, no table rewrite, no lock held over the
row count.

Reach for something heavier only when you actually need `NOT NULL`:

- **Column needs `NOT NULL` and a constant default** — `ADD COLUMN x TYPE
  NOT NULL DEFAULT <value>` in one statement. Modern Postgres (11+) doesn't
  rewrite the table for a constant default, so this is still cheap.
- **Column needs `NOT NULL` but the value must be computed per row** — add
  it nullable first, backfill with `UPDATE` (see `references/backfills.md`),
  then a second migration adds the `NOT NULL` constraint once every row is
  populated. Don't do all three in one migration if the backfill runs
  against a large table — split so the constraint-add migration is a fast,
  separate step.

## Down reverses Up exactly

`DROP COLUMN closed_at` undoes `ADD COLUMN closed_at` — nothing more,
nothing less. If Up also backfills or adds a constraint, Down must undo all
of it in reverse order (drop the constraint, then the column), not just the
column add. `just migrate-down && just migrate-up` (step 4 in `SKILL.md`)
is how you prove this rather than eyeball it — a Down that silently leaves
a constraint or index behind still "succeeds" but breaks the next
`migrate-up`.

## Renames

No rename migration exists yet in this repo to cite, but the same
symmetry rule applies: `ALTER TABLE t RENAME COLUMN old TO new` going up
pairs with `ALTER TABLE t RENAME COLUMN new TO old` going down. A rename
is also the one alteration that breaks callers immediately — check
`internal/services/*/store/queries/*.sql` for every reference to the old
column name before renaming, since sqlc will fail to generate (not silently
skip) if a query still refers to a column that's gone after step 5 in
`SKILL.md`.
