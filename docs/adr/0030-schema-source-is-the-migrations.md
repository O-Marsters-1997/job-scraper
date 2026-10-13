# ADR 0030 — The migrations are the schema source

Issue: [#874](https://github.com/O-Marsters-1997/job-scraper/issues/874). The postgres-best-practices skill
and go-idiomatic (`secrets.md`, config) are the authority. Implemented in W2 onward.

## Decision

- **Migrations are embedded** with `embed.FS` and run through `goose.NewProvider`. Nothing depends on the
  working directory, and `os.Chdir` in `pgtest` and `RunMigrations` goes.
- **sqlc reads `scripts/migrations` directly.** `internal/data/db/sqlc/schema.sql` and the rule to mirror
  every migration into it by hand are deleted. `just generate` and CI's `sqlc generate && git diff
  --exit-code` stay.
- **Applied migrations are still never edited.** A change is a new migration (`just migrate-create`).

## Consequences

- One source of truth, so schema and migrations cannot drift.
- Binaries and tests run from any directory.
- sqlc must understand every migration statement, so a migration using unsupported DDL needs a
  sqlc-friendly shape or an override.
