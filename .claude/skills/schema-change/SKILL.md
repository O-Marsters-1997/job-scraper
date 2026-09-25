---
name: schema-change
description: Change the Postgres schema: create, alter or drop tables, columns, indexes or constraints, backfill data, and update the sqlc queries and generated code to match. Use for "write a migration", "change the schema", "add a column", "new index", "rename a field", "backfill".
paths: ["scripts/migrations/**", "internal/data/sqlc/**", "internal/data/db/**"]
---

# Schema change

A schema change touches four layers that don't auto-sync: the migration, the
hand-maintained `schema.sql` mirror, the sqlc-generated Go, and the wrapper
code around it. Do them in this order — skipping one leaves the layers
inconsistent in a way tests won't catch until CI's `git diff --exit-code`.

1. **Check state.** `just up` starts Postgres, then `just migrate-status`
   shows what's already applied.

2. **Create the migration.** `just migrate-create <verb_noun>` (e.g.
   `just migrate-create add_index_on_url`). Never edit a migration that's
   already been applied — add a new one instead
   (`AGENTS.md` "Gotchas"). See `references/ordering.md` if this migration
   depends on another one that hasn't merged yet.

3. **Write the SQL.** `-- +goose Up` and `-- +goose Down`; Down must reverse
   Up exactly (drop what Up created, in reverse order). Conventions —
   column types, naming, constraints — are in `references/sql-conventions.md`.
   Altering an existing table (nullable-first backfills, renames, drops)?
   Read `references/altering-tables.md`. Migrating or reconciling existing
   rows? Read `references/backfills.md`.

4. **Prove it's reversible.** `just migrate-up`, then
   `just migrate-down && just migrate-up`. If Down errors or Up-after-Down
   fails, Down doesn't mirror Up — fix it now, not after this ships.

5. **Mirror the end state into `internal/data/sqlc/schema.sql` by hand.**
   sqlc reads only this file, never `scripts/migrations/`
   (`AGENTS.md` "sqlc"). Add/change table and index definitions to match
   what the migration produces after Up.

6. **Update the queries.** Edit or add to
   `internal/data/sqlc/queries/<table>.sql`. See
   `references/sqlc-queries.md` for which annotation
   (`:one`/`:many`/`:exec`/`:execrows`/`:batchexec`) fits, and
   `sqlc.arg`/`sqlc.narg` usage.

7. **Regenerate.** Confirm your local `sqlc version` matches the version
   pinned in `.github/workflows/ci.yml` (`sqlc-dev/sqlc/cmd/sqlc@v1.31.1` as
   of writing — check the file, it drifts), then run `just generate`. This
   rewrites `internal/data/db/pgsqlc/**` — never hand-edit that tree
   (`AGENTS.md` "Boundaries").

8. **Update the wrappers.** In `internal/data/db/*.go`, wrap generated calls:
   map `pgx.ErrNoRows` → `providers.ErrNotFound`, wrap other errors as
   `fmt.Errorf("db.Method: %w", err)`. Copy the pattern in
   `internal/data/db/profile.go`. Wherever a signature changed, update the
   matching provider interface and mock in `internal/data/providers/`
   (copy `internal/data/providers/mock_profile.go`). Error-mapping detail,
   including whether a unique-violation sentinel exists, is in
   `references/sqlc-queries.md`.

9. **Verify.** Needs Docker (`just up`):
   ```
   sqlc generate && git diff --exit-code internal/data/db/pgsqlc && go test ./internal/data/db/...
   ```
   DB tests use a real Postgres testcontainer via `testDB` in
   `internal/data/db/db_test.go` — don't start a second container. Run
   `go test` from the repo root; migrations resolve `scripts/migrations`
   relative to CWD.

## References

- `references/sql-conventions.md` — column types, index naming, `CHECK`
  constraints, `ON DELETE CASCADE`.
- `references/altering-tables.md` — adding/dropping/renaming columns on an
  existing table, keeping Down symmetric.
- `references/backfills.md` — `INSERT ... SELECT`, `DISTINCT ON`, a
  dead-letter table for rows that can't migrate cleanly.
- `references/ordering.md` — why migration timestamps matter across
  branches and how to fix a collision.
- `references/sqlc-queries.md` — annotation choice, `sqlc.arg`/`sqlc.narg`,
  and the actual error-mapping convention in this codebase.
