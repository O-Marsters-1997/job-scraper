---
name: new-handler
description: Add an HTTP handler/endpoint and its frontend api function and hook. Covers dto, service, module route, zod schema, mock and query hook. Use for "new handler", "add an endpoint", "new route", "expose X to the frontend".
paths: ["internal/services/**", "internal/handlers/**", "internal/api/**", "internal/dto/**", "frontend/src/api/**", "frontend/src/hooks/**", "frontend/src/mocks/**", "frontend/src/types/**"]
---

# New Handler

An endpoint touches four backend pieces and four frontend ones. Work top to bottom.

## 0. Which context?

Find the owning context in [ADR 0011](../../../docs/adr/0011-modular-monolith-by-context.md)
(`jobsearch`, `scoring`, `applications`, `cvtemplates`, `identity`). Pick it by which tables the
endpoint *writes*. Reads may join other contexts' tables. If the route would write two contexts'
tables, it belongs to one of them and calls the other's tx-scoped port. Nothing fits? Stop and ask.

If the endpoint needs a new table, column or query, run the `schema-change` skill first. Come
back once the store method exists.

## Backend

Every handler is built from `Handle(decode, call, respond)` in `internal/handlers`, directly or
through a CRUD generic ([ADR 0008](../../../docs/adr/0008-handlers-over-feature-services.md)).

### 1. dto input type

Only if the body or response is new. Add it to `internal/dto/`: a plain struct with JSON tags
and no business logic. `dto` holds wire shapes and types that cross a facade; a shape used only
inside the context goes in that context's own package. If a path ID belongs on it (anything but
a plain `GetByID`/`Delete`), tag that field `json:"-" path:"id"` (or `path:"docId"`/`path:"tabId"`
for a two-segment route), as in `SetCompanyTrackingInput`. The generic wrapper fills the field
from the chi URL param after decoding the body, so a request body can never set it.

### 2. Service

The context's main feature lives in the context root, `internal/services/<ctx>/service.go`, with
constructor `NewService`. Any other feature gets its own package, `internal/services/<feature>/`
(e.g. `applicationstatuses` in `applications`). A feature package never imports the store; the
module's `New` passes the store in. All constructor args are required:

- a `store` interface declared in this package, listing only the store methods it calls
- small local interfaces for anything else: queue publisher, verifier, or another context's
  facade (declared here and satisfied by that `*Module`)

No `With*` setters, and no concrete type from another context.

Match the method's signature to the generic wrapper it will bind to (see the table in step 3).
Return `(dto.Y, error)`, where the error is an `apperr` kind:

| Kind | Status |
|---|---|
| `Invalid` | 400 |
| `Unauthorized` | 401 |
| `NotFound` | 404 |
| `Conflict` | 409 |
| `Unprocessable` | 422 |
| `Upstream` | 502 |
| `Unavailable` | 503 |

To add an extra field to the error body, wrap it:
`apperr.WithFields(apperr.Conflict(...), map[string]any{"count": n})`. If there's no rule or
orchestration, don't write a service at all; see `references/pass-through-routes.md`.

Test it in the same package with a hand-written fake of the `store` interface. There are no
generated mocks.

### 3. Wire and route

Construct the service in the context's `New` (`internal/services/<ctx>/module.go`), next to its
siblings. That is the only place it's built. `cmd/api/main.go` builds modules, never services.

Add the route in the module's `Routes` (`internal/services/<ctx>/routes.go`), binding it to a generic
wrapper from `internal/handlers/generic.go`:

| Wrapper | Your method's shape | Status |
|---|---|---|
| `handlers.GetAll` | `(ctx, userID) (Out, error)` | 200 |
| `handlers.GetByID` | `(ctx, userID, id) (Out, error)` | 200 |
| `handlers.Query[Q]` | `(ctx, userID, q Q) (Out, error)` | 200 |
| `handlers.Create` | `(ctx, userID, in In) (Out, error)` | 201 |
| `handlers.Update` | `(ctx, userID, in In) (Out, error)` | 200 |
| `handlers.Delete` | `(ctx, userID, id) error` | 204 |

- To get 204 instead of the verb's default, return `struct{}` as `Out` on `Create`/`Update`. Use
  this for a bodyless action, or a write with nothing to send back.
- An action reuses whichever wrapper matches its shape, whatever the HTTP method.
  `POST .../scrape` is still a `GetByID`.
- `Query` needs a query dto with string fields and JSON tags matching the query keys. The
  service parses and validates them.

The route line is exactly `r.<Method>("path", handlers.<Wrapper>(m.svc.Method))`: no `func`
literal, no status argument, no inline validation. If the route sets cookies, redirects, streams
or uses service-token auth, use `references/non-adapter-routes.md` instead of the table above.

A brand-new context also needs building in `cmd/api/main.go` and passing to `api.NewRouter`. An
existing context is already mounted.

## Frontend

### 4. Type

`frontend/src/types/<x>.ts`: the shared type, not declared inline in the api file.

### 5. api function + zod schema

Copy `frontend/src/api/applicationStatuses.ts` for the function shape: one exported async
function per operation, the `useMocks()` branch first and the `apiFetch` call second. Copy
`frontend/src/api/scores.ts` for the zod schema pattern: `apiFetch(path, init, schema)` parses
the response rather than casting it.

### 6. Mocks

Add a `useMocks()` branch to each new api function (see step 5), backed by fake rows in
`frontend/src/mocks/db.ts`. This is required if an e2e test covers the page, because e2e runs
with `VITE_MOCK=true` and no backend.

### 7. Hook

Copy `frontend/src/hooks/useProfile.ts`: `queryOptions` + `createQuery` for reads, and
`createMutation` with `queryClient.invalidateQueries` for writes.

## Verify

```
go test ./internal/services/... ./internal/handlers/...
cd frontend && bun run typecheck && bun run test && bunx playwright test
```

`bun run test` runs every `*.check.ts` file (there's no test framework). `bunx playwright test`
is the e2e suite. Use bun only, never npm/pnpm/yarn (`frontend/AGENTS.md`).
