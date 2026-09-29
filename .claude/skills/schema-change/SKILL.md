---
name: schema-change
description: Change the Postgres schema: create, alter or drop tables, columns, indexes or constraints, backfill data, and update the sqlc queries and generated code to match. Use for "write a migration", "change the schema", "add a column", "new index", "rename a field", "backfill".
---

# Schema change

A schema change touches four layers that don't auto-sync: the migration, the
hand-maintained `schema.sql` mirror, the sqlc-generated Go, and the store
code around it. Do them in this order — skipping one leaves the layers
inconsistent in a way tests won't catch until CI's `git diff --exit-code`.

0. **Find the owner.** Every table belongs to exactly one context
   ([ADR 0011](../../../docs/adr/0011-modular-monolith-by-context.md)). A new
   table goes to the context whose rules write it; if none fits, stop and ask.
   Only the owner's store writes the table. Other contexts may read-join it.

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

5. **Mirror the end state into `internal/data/db/sqlc/schema.sql` by hand.**
   sqlc reads only this file, never `scripts/migrations/`
   (`AGENTS.md` "sqlc"). Add/change table and index definitions to match
   what the migration produces after Up.

6. **Update the queries.** Edit or add to the owning context's
   `internal/services/<ctx>/store/queries/<table>.sql`. A query that writes a
   table belongs in that table's owner; a read-join may live in any context's
   store. See
   `references/sqlc-queries.md` for which annotation
   (`:one`/`:many`/`:exec`/`:execrows`/`:batchexec`) fits, and
   `sqlc.arg`/`sqlc.narg` usage.

7. **Regenerate.** Confirm your local `sqlc version` matches the version
   pinned in `.github/workflows/ci.yml` (`sqlc-dev/sqlc/cmd/sqlc@v1.31.1` as
   of writing — check the file, it drifts), then run `just generate`. This
   rewrites every generated tree (`internal/services/<ctx>/store/sqlc/**`);
   never hand-edit them (`AGENTS.md` "Boundaries"). A context's first query
   also needs its own `sql:` block in `sqlc.yaml` pointing at the shared
   `schema.sql`, and a `<ctx>-store` depguard rule in `.golangci.yml` (copy
   `applications-store`) so only `internal/services/<ctx>/` can import the
   store.

8. **Update the store.** In `internal/services/<ctx>/store/store.go`, wrap
   generated calls. Map `pgx.ErrNoRows` to the store's own `ErrNotFound`,
   and wrap other errors as `fmt.Errorf("store.Method: %w", err)`. Return
   `dto` or store-local types, never generated sqlc structs. Every sqlc row
   → `dto` converter lives in `store/transform.go`, named `to<Name>DTO` after
   the dto it builds (`toApplicationDTO`, `toApplicationStatusDTO`). Wherever a
   signature changed, update the consuming service's own `store`
   interface and its hand fake. There is no shared mock package.
   Error-mapping detail is in `references/sqlc-queries.md`.

   If the change adds a side effect in another context's tables inside the
   same transaction, don't write those tables. Call the owner's tx-scoped
   port instead, e.g. `scoring.JobsChanged(ctx, tx, jobIDs)`.

9. **Verify.** Needs Docker (`just up`):
   ```
   sqlc generate && git diff --exit-code && go test ./internal/services/...
   ```
   Store tests use a real Postgres testcontainer via `internal/pgtest`. Run
   `go test` from the repo root, because migrations resolve
   `scripts/migrations` relative to CWD.

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
  and the store error-mapping convention.
