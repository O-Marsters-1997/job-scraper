---
name: new-handler
description: Add an HTTP handler/endpoint and its frontend api function and hook. Covers dto, service, route, zod schema, mock and query hook. Use for "new handler", "add an endpoint", "new route", "expose X to the frontend".
paths: ["internal/handlers/**", "internal/router.go", "internal/services/**", "internal/dto/**", "frontend/src/api/**", "frontend/src/hooks/**", "frontend/src/mocks/**", "frontend/src/types/**"]
---

# New Handler

An endpoint touches five backend pieces and four frontend ones. Work top to bottom.

## 0. New persistence?

If the endpoint needs a new table/column/query, run the `schema-change` skill first — it
covers the migration and sqlc regen. Come back here once the provider method exists.

## Backend

The shape below is current: [ADR 0020](../../../docs/adr/0020-handlers-as-http-adapter-over-feature-services.md)
and [ADR 0021](../../../docs/adr/0021-services-directory-and-crud-generics.md) are both rolled
out. `internal/handlers` holds only `adapter.go`, `generic.go` and the misfit files (auth,
google, ingest, cv export) — no handler struct owns your route.

### 1. dto input type

Add the input type to `internal/dto/` — plain struct, JSON tags, no business logic. If a path
ID belongs on it (anything but a plain `GetByID`/`Delete`), tag that field
`json:"-" path:"id"` (or `path:"docId"`/`path:"tabId"` for a two-segment route) — see
`internal/dto/company_input.go`'s `SetCompanyTrackingInput` for the pattern. The generic
wrapper fills it from the chi URL param after decoding the body, so a request body can never
set it.

### 2. Service

Package `internal/services/<feature>` (e.g. `internal/services/companies`,
`internal/services/sourcetargets`) — not bare `internal/<feature>`. The exception:
`internal/candidates` and `internal/ingest` stay where they are, because the worker/scraper
import them too; only a service that exists purely to back an HTTP route goes under
`internal/services/`.

Constructor args are all required — `providers.X` interfaces (already in
`internal/data/providers/`) for persistence, small interfaces declared in the service's own
package for anything else (queue publisher, verifier, scorer). No `With*` setters.

Match your method's signature to whichever generic wrapper it'll bind to (step 3) — see that
table. Return `(dto.Y, error)`, the error being an `apperr` kind: `Invalid` (400),
`Unauthorized` (401), `NotFound` (404), `Conflict` (409), `Unprocessable` (422), `Upstream`
(502), `Unavailable` (503). Need an extra field in the error body (e.g. a count)? Wrap it:
`apperr.WithFields(apperr.Conflict(...), map[string]any{"count": n})` — the adapter merges it
into `{"error": ...}` automatically. If there's no rule or orchestration to add, don't write a
service at all; see `references/pass-through-routes.md`.

### 3. Route + wiring

`internal/services.go` builds every service once in `newServices(db, q, creds) *services`.
Add your service's field and construction there, next to its siblings — this is the only place
that constructs it.

`internal/router.go` adds the chi route, binding it to a generic wrapper from
`internal/handlers/generic.go`:

| Wrapper | Your method's shape | Status |
|---|---|---|
| `handlers.GetAll` | `(ctx, userID) (Out, error)` | 200 |
| `handlers.GetByID` | `(ctx, userID, id) (Out, error)` | 200 |
| `handlers.Query[Q]` | `(ctx, userID, q Q) (Out, error)` | 200 |
| `handlers.Create` | `(ctx, userID, in In) (Out, error)` | 201 |
| `handlers.Update` | `(ctx, userID, in In) (Out, error)` | 200 |
| `handlers.Delete` | `(ctx, userID, id) error` | 204 |

Return `struct{}` as `Out` on `Create`/`Update` to get 204 instead of the verb's default (a
bodyless action, or a write with nothing to send back). An action reuses whichever wrapper
matches its shape regardless of HTTP method — `POST .../scrape` is still a `GetByID`. `Query`
needs a query dto (step 1's sibling: string fields, json tags matching the query keys); the
service parses and validates them.

The line in `router.go` is exactly `r.<Method>("path", handlers.<Wrapper>(svc.Method))` — no
`func` literal, no status argument, no inline validation. If your route sets cookies,
redirects, streams, or uses service-token auth, see `references/non-adapter-routes.md` instead
of the table above.

`cmd/api/main.go` only builds top-level infra (db, queue, credstore) and calls
`app.NewRouter(db, q, cs)` — never wire a service there.

## Frontend (ready now — these are real, current patterns)

### 4. Type

`frontend/src/types/<x>.ts` — the shared type, not declared inline in the api file.

### 5. api function + zod schema

Copy `frontend/src/api/applicationStatuses.ts` for the function shape (one exported async
function per operation, `useMocks()` branch first, `apiFetch` call second) and
`frontend/src/api/scores.ts` for the zod schema pattern (`apiFetch(path, init, schema)` —
parse the response, don't just cast it).

### 6. Mocks

Add a `useMocks()` branch to each new api function (see step 5) backed by fake rows in
`frontend/src/mocks/db.ts`. Required if the page is covered by an e2e test — e2e runs with
`VITE_MOCK=true` and no backend.

### 7. Hook

Copy `frontend/src/hooks/useProfile.ts` — `queryOptions` + `createQuery` for reads,
`createMutation` with `queryClient.invalidateQueries` for writes.

## Verify

```
go test ./internal/handlers/... ./internal/services/<feature>/...
cd frontend && bun run typecheck && bun run test && bunx playwright test
```

`bun run test` runs every `*.check.ts` file (no test framework); `bunx playwright test` is the
e2e suite. Use bun only — never npm/pnpm/yarn (`frontend/AGENTS.md`).
