# sqlc queries and wrapper error handling

Counted directly in `internal/data/sqlc/queries/*.sql` (18 files) at time
of writing — recount if this feels stale, sqlc will tell you immediately
if an annotation is wrong:

| Annotation    | Count | Use for |
|---------------|-------|---------|
| `:one`        | 47    | Query returns exactly one row (get-by-id, insert...returning, update...returning). Errors with `pgx.ErrNoRows` if nothing matches. |
| `:many`       | 25    | Query returns zero or more rows into a slice. |
| `:exec`       | 29    | Statement returns nothing you need — delete, update with no `RETURNING`, insert with no `RETURNING`. |
| `:execrows`   | 7     | Same as `:exec` but you need the affected row count (e.g. to detect a no-op update and return `providers.ErrNotFound` if it's 0). |
| `:batchexec`  | 1     | Same statement run for a batch of parameter sets in one round trip — `internal/data/sqlc/queries/jobs.sql`'s `UpsertJobs`, used for bulk job ingest. Reach for this only when you're already writing N copies of the same `:exec` in a loop. |

## sqlc.arg vs sqlc.narg

`sqlc.arg(name)` is a required parameter — sqlc generates a plain typed
field. `sqlc.narg(name)` ("nullable arg") generates a `pgtype.X` wrapper so
the caller can pass SQL `NULL`. `internal/data/sqlc/queries/canonical_jobs.sql`
has both in the same query: `sqlc.narg(board_id)`/`sqlc.narg(posting_id)`
because a job might not be tied to a known board yet, alongside plain
`sqlc.arg(title)`, `sqlc.arg(url)`, etc. for fields that are always present.
Use `narg` only for a column that's genuinely optional on write — reach for
`arg` by default.

## Wrapper error mapping

Every `internal/data/db/*.go` wrapper follows `profile.go`'s pattern:

```go
row, err := db.queries.GetUserProfile(ctx, uid)
if errors.Is(err, pgx.ErrNoRows) {
    return dto.Profile{}, providers.ErrNotFound
}
if err != nil {
    return dto.Profile{}, fmt.Errorf("db.GetProfile: %w", err)
}
```

`providers.ErrNotFound` is declared once, in `internal/data/providers/application.go`,
and every provider mock (e.g. `mock_profile.go`, `mock_company.go`,
`mock_search_config.go`) returns the same sentinel on a miss — that's what
lets handler code check `errors.Is(err, providers.ErrNotFound)` regardless
of whether it's talking to Postgres or a test mock.

**Unique-violation (23505) handling exists, but it's not a shared sentinel
in the `db` package.** Three call sites check the Postgres code directly —
`internal/data/db/auth.go:67`, `internal/api/handlers/source_targets.go:162`,
`internal/api/handlers/applications.go:80` — each with its own
`errors.As(err, &pgErr) && pgErr.Code == "23505"` check inline, mapped to
whatever conflict error fits that call site (not a single
`providers.ErrConflict` reused everywhere). If your new query can violate a
`UNIQUE` constraint and the caller needs to distinguish that from other
failures, follow the existing per-site pattern — don't invent a new shared
sentinel unless you're deliberately consolidating the three.
