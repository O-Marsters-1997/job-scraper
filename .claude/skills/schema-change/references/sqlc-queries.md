# sqlc queries and wrapper error handling

Counted across `internal/services/applications/store/queries/*.sql` (ADR 0011) —
recount if this feels stale, sqlc will tell you immediately if an annotation
is wrong:

| Annotation    | Count | Use for |
|---------------|-------|---------|
| `:one`        | 47    | Query returns exactly one row (get-by-id, insert...returning, update...returning). Errors with `pgx.ErrNoRows` if nothing matches. |
| `:many`       | 25    | Query returns zero or more rows into a slice. |
| `:exec`       | 29    | Statement returns nothing you need — delete, update with no `RETURNING`, insert with no `RETURNING`. |
| `:execrows`   | 7     | Same as `:exec` but you need the affected row count (e.g. to detect a no-op update and return the store's `ErrNotFound` if it's 0). |
| `:batchexec`  | 1     | Same statement run for a batch of parameter sets in one round trip — `jobs.sql`'s `UpsertJobs`, used for bulk job ingest. Reach for this only when you're already writing N copies of the same `:exec` in a loop. |

## sqlc.arg vs sqlc.narg

`sqlc.arg(name)` is a required parameter — sqlc generates a plain typed
field. `sqlc.narg(name)` ("nullable arg") generates a `pgtype.X` wrapper so
the caller can pass SQL `NULL`. `canonical_jobs.sql`
has both in the same query: `sqlc.narg(board_id)`/`sqlc.narg(posting_id)`
because a job might not be tied to a known board yet, alongside plain
`sqlc.arg(title)`, `sqlc.arg(url)`, etc. for fields that are always present.
Use `narg` only for a column that's genuinely optional on write — reach for
`arg` by default.

## Store error mapping

Every store method wraps its generated call like this (`internal/services/applications/store/store.go`):

```go
row, err := s.queries.GetApplication(ctx, sqlc.GetApplicationParams{UserID: uid, ID: aid})
if errors.Is(err, pgx.ErrNoRows) {
    return dto.Application{}, ErrNotFound
}
if err != nil {
    return dto.Application{}, fmt.Errorf("get application: %w", err)
}
```

Each context's store declares its own sentinels in its own package:

- `ErrNotFound = apperr.NotFound("not found")`, plus specific ones where the caller must tell
  them apart (e.g. `ErrTabNotFound`).
- Use an `apperr` kind when the sentinel maps to an HTTP status.
- Services in the same context match with `errors.Is(err, store.ErrNotFound)`.
- The context's root package re-exports a sentinel only when another context or the worker must
  match it.

**Unique violations (23505)** are checked in the store at the call site, with
`errors.As(err, &pgErr) && pgErr.Code == "23505"`, and mapped to a sentinel that fits that call:

| Call site | Sentinel |
|---|---|
| `internal/services/identity/store/store.go` (`CreateUser`) | `ErrUsernameTaken` |
| `internal/services/applications/store/store.go` (`CreateApplication`) | `ErrApplicationExists` |
| `internal/services/jobsearch/store/store.go` (`CreateSourceTarget`, `CreateSourceTargetWithRun`) | `ErrSourceTargetExists` |

If your new query can violate a `UNIQUE` constraint and the caller needs to tell that apart from
other failures, follow the same per-site pattern. Don't add a shared conflict sentinel.
