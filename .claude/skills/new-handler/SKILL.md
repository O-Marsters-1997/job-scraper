---
name: new-handler
description: Add an HTTP handler/endpoint and its frontend api function and hook. Covers dto, service, route, zod schema, mock and query hook. Use for "new handler", "add an endpoint", "new route", "expose X to the frontend".
paths: ["internal/handlers/**", "internal/router.go", "internal/dto/**", "frontend/src/api/**", "frontend/src/hooks/**", "frontend/src/mocks/**", "frontend/src/types/**"]
---

# New Handler

An endpoint touches five backend pieces and four frontend ones. Work top to bottom.

## 0. New persistence?

If the endpoint needs a new table/column/query, run the `schema-change` skill first — it
covers the migration and sqlc regen. Come back here once the provider method exists.

## Backend

**ADR 0020 status check (verified by this skill, re-check if it's been a while):**
`internal/apperr` does not exist yet, and no handler uses `User`/`ID`/`Body`/`BodyID` or the
`Caller`/`DecodeJSON`/`WriteJSON`/`WriteError` helpers — every handler still decodes,
validates and writes responses inline with raw `net/http` (see `internal/handlers/profile.go`,
`internal/handlers/companies.go`). Steps 2-4 below describe the **target** shape from
[ADR 0020](../../../docs/adr/0020-handlers-as-http-adapter-over-feature-services.md)
(accepted, rollout pending). There is no canonical file to copy for these steps —
do **not** copy an existing `internal/handlers/*.go` file as the pattern, it's pre-ADR.

### 1. dto input type

Add the input type to `internal/dto/` — plain struct, no business logic. Existing dto types
(e.g. `internal/dto/company_upsert.go`) show the shape convention; the ADR adds JSON tags to
input types as the rollout touches each one, so add tags on your new type even though older
ones don't have them yet.

### 2. Service (target shape, pending ADR 0020)

Package `internal/<feature>` (e.g. `internal/companies`, `internal/sourcetargets`).
Constructor args are all required — `providers.X` interfaces (already in
`internal/data/providers/`) for persistence, small interfaces declared in the service's own
package for anything else (queue publisher, verifier, scorer). No `With*` setters.

Methods take `(ctx, userID, id…, in dto.X)` — the caller's user ID and path IDs are always
service args, never dto fields — and return `(dto.Y, error)`, the error being an `apperr` kind:
`Invalid` (400), `Unauthorized` (401), `NotFound` (404), `Conflict` (409), `Unprocessable`
(422), `Upstream` (502), `Unavailable` (503). Add a service method only when there's a rule or
orchestration to hold; see `references/pass-through-routes.md` if there isn't one.

### 3. Route + wiring in `internal/router.go`

This repo wires everything in one place: `NewRouter` in `internal/router.go` both constructs
handlers next to their siblings (`profileH := handlers.NewProfileHandler(db)`) and adds the
chi route in the matching `r.Route(...)` block. `cmd/api/main.go` only builds top-level infra
(db, queue, credstore) and calls `app.NewRouter(db, q, cs)` — there is no per-handler wiring
there, wire your service/handler in `router.go`, not `main.go`.

The chi route line itself is ordinary code today and after the ADR alike — copy the
mechanics from a neighbouring route. What it points to is what's pending: today a handler
struct method (pre-ADR, raw `net/http`); after rollout, an adapter wrapper call. If your
route sets cookies, redirects, streams, or uses service-token auth, see
`references/non-adapter-routes.md` instead.

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
go test ./internal/handlers/ ./internal/<feature>/
cd frontend && bun run typecheck && bun run test && bunx playwright test
```

`bun run test` runs every `*.check.ts` file (no test framework); `bunx playwright test` is the
e2e suite. Use bun only — never npm/pnpm/yarn (`frontend/AGENTS.md`).
