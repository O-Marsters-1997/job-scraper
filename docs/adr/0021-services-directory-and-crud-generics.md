# ADR 0021 — Feature services under internal/services; CRUD-shaped generics replace User/ID/Body/BodyID

**Status:** Accepted. Its "misfits stay in `internal/handlers`, as functions over a service, not
a struct" clause is partially superseded by
[ADR 0022](0022-handle-as-the-one-handler-pipeline.md): misfits still live there as
`func(svc) http.HandlerFunc` closures, but now build that closure from the same `Handle`
function the CRUD generics use.

## Context

ADR 0020's rollout landed the adapter (`Caller`, `DecodeJSON`, `WriteJSON`, `WriteError`) and
the `User`/`ID`/`Body`/`BodyID` wrappers, and moved orchestration into feature packages
(`internal/companies`, `internal/sourcetargets`, ...). Two problems showed up once every route
was actually migrated:

- Feature service packages sat directly under `internal/` next to domain packages the worker
  and scraper also import (`internal/candidates`, `internal/ingest`), with no boundary marking
  which ones exist only to back an HTTP route.
- `User`/`ID`/`Body`/`BodyID` cover the four shapes ADR 0020 anticipated, but real routes needed
  more: singleton reads/writes (`/profile`, `/ai-prefs`), query-string-driven reads (`/jobs`,
  `/applications`), a second path ID (`/tracked-docs/{docId}/tabs/{tabId}`), and a 409 that
  carries an extra field (`count` when a status is in use). `router.go` absorbed the mismatch as
  inline closures — a validated, multi-provider-calling function defined at the route, e.g.
  `/application-statuses` POST/PATCH and `/companies/{id}/boards` POST — which is exactly the
  per-feature orchestration ADR 0020 meant to keep out of the adapter layer.

## Decision

- **`internal/services/<feature>`.** Every service that exists only to back an HTTP route moves
  under `internal/services/`: `companies`, `applications`, `applicationstatuses`, `aiprefs`,
  `aicredentials`, `scoringconfig`, `sourcetargets`, `jobreasoning`, `cvtemplates`, `jobs`,
  `sources`, `profile`, `auth`, `google`. `internal/candidates` and `internal/ingest` stay where
  they are: the worker and scraper import them too, so they're domain packages that HTTP happens
  to use, not HTTP-only services. `ATSBoardVerifier` (a `companies.BoardVerifier` implementation)
  moves into `internal/services/companies` with the interface it implements.
- **Generics keyed by shape, not verb count.** `internal/handlers/generic.go` replaces
  `User`/`ID`/`Body`/`BodyID` with `GetAll`, `GetByID`, `Query[Q]`, `Create`, `Update`, `Delete`,
  each fixing one function shape and one status:

  | Wrapper | Function shape | Status |
  |---|---|---|
  | `GetAll[Out]` | `(ctx, userID) (Out, error)` | 200 |
  | `GetByID[Out]` | `(ctx, userID, id) (Out, error)` | 200 |
  | `Query[Q, Out]` | `(ctx, userID, q Q) (Out, error)` | 200 |
  | `Create[In, Out]` | `(ctx, userID, in In) (Out, error)` | 201 |
  | `Update[In, Out]` | `(ctx, userID, in In) (Out, error)` | 200 |
  | `Delete` | `(ctx, userID, id) error` | 204 |

  An `Out` of `struct{}` always writes 204, overriding the verb's default — this is how a
  bodyless action (tab hide/show) or a body-only update with no response (`/profile`,
  `/ai-credentials`) reaches 204 without its own wrapper. An action reuses whichever verb
  matches its shape regardless of HTTP method: `POST /source-targets/{id}/scrape` and
  `POST /jobs/{id}/reasoning` both go through `GetByID`, `POST /scores/rescore` goes through
  `GetAll`. `Caller`/`DecodeJSON`/`WriteJSON`/`WriteError` become private (`caller`,
  `decodeBody`/`decodeQuery`, `writeJSON`, `writeError`); nothing outside `internal/handlers`
  called them directly.
- **Path IDs travel on the input dto**, tagged `path:"name"` (e.g. `path:"id"`, or `path:"docId"`
  / `path:"tabId"` for a two-segment route), filled from chi URL params by `Create`/`Update`
  after JSON-decoding the body — so the body can never set them, same guarantee ADR 0020's
  `(ctx, userID, id…, in)` ordering gave. `GetByID` and `Delete` still take `id` as a plain
  argument; they have no body to smuggle an override through.
- **`Query[Q]`** decodes the URL query string into `Q` by flattening `url.Values` into
  `map[string]string` and JSON-decoding that into `Q`. `Q`'s fields stay strings with matching
  json tags; the service parses and validates them (`/jobs`'s `limit`/`availability`/`cursor`,
  `/applications`'s `status_id`, `/applications/for-jobs`'s comma-separated `job_ids`,
  `/sources/resolve`'s `url`).
- **`apperr.WithFields`/`FieldsFor`** attach extra fields to a returned error, merged into the
  `{"error": ...}` body. This replaces the one-off `map[string]any{"error": ..., "count": ...}`
  response `DeleteApplicationStatus` used to write by hand; it's now
  `apperr.WithFields(apperr.Conflict(...), map[string]any{"count": n})` returned from
  `applicationstatuses.Service.Delete`, mapped by the same adapter path as any other error.
- **A route with no logic binds a provider method value directly**, same as ADR 0020 (e.g.
  `handlers.GetAll(db.ListCompaniesForUser)`). The moment it would need a closure — reordering
  arguments, wrapping the output, adding a check — that closure becomes a named method on a
  service instead, even a trivial one. `router.go` contains route lines only: no `func` literals,
  no inline validation, no status numbers (the verb fixes the status; `struct{}` overrides it).
- **A tiny service is fine when its only job is to satisfy a shape.** `services/sources.List`
  and `.Resolve` wrap the stateless `registry.Sources()`/`detect.ResolveBoard()` calls so they
  can bind through `GetAll`/`Query` instead of a closure. `services/aicredentials.Update` holds
  the `provider required` check and the save-vs-delete branch that PUT `/ai-credentials` needs.
  These aren't premature abstraction — they exist because ADR 0020's own router already had this
  exact logic as a closure, and this ADR's rule is that no such closure survives in `router.go`.
- **Composition root.** `internal/services.go` (package `app`) builds every feature service once
  in `newServices(db, q, creds) *services` and hands `NewRouter` a `*services` struct to route
  against. `NewRouter`'s body is `NewRouter(db, q, creds) http.Handler { ...; svc := newServices(...); ...routes... }`
  — construction and routing are visibly separate, and `cmd/api/main.go` is unaffected.
- **Misfits stay in `internal/handlers`, as functions over a service, not a struct.** Cookie-
  setting, redirecting, streaming and service-token routes (`Login`/`Signup`/`Logout`,
  `OAuthStart`/`OAuthCallback`, `ExportCV`, `Ingest`/`IngestBatch`) become
  `func X(svc) http.HandlerFunc` closures constructed once in `router.go`, replacing the old
  `NewXHandler(...) *XHandler` structs. Their password/session/OAuth-token logic moves into
  `services/auth` and `services/google`; `services/cvtemplates.ExportPDF` and
  `internal/ingest.Ingester` supply the rest. `GET /google/status` and `DELETE /google/link` turn
  out not to be misfits at all once the logic moves to `services/google` — they bind through
  `GetAll`/`Delete` like any other route; only the redirect-driven `oauth/start` and
  `oauth/callback` stay hand-written.
- **Wire changes.** `POST /companies/{id}/boards` moves from 200 to 201 (it's a `Create`);
  `POST /tracked-docs` moves from 201 to 204 (`Create` with a `struct{}` output — nothing in
  the response to send back); `POST /source-targets/{id}/scrape` moves from 202 to 200 (it's a
  `GetByID`, not its own verb). Tab hide/show were already 204 under `ID2` and stay 204 under
  `Update`. No response body shape changes. The frontend only branches on `res.ok`/specific
  error codes for these routes, so none of them needed a client-side change.

## Consequences

- Adding a plain CRUD route is a dto (with `path` tags where it has path IDs), a service method,
  and one `router.go` line choosing the matching verb — no new adapter wrapper needed unless the
  shape is genuinely new.
- `internal/services/<feature>` is a browsable list of "what backs the API" distinct from
  `internal/candidates`, `internal/ingest`, and the scraper/source packages the worker owns.
- `router.go` is auditable at a glance: every line is `r.<Method>(path, handlers.<Verb>(svc.X))`,
  so a misrouted method, status, or missing auth group stands out instead of hiding in a closure.
- `struct{}` as a signal ("this writes 204") is implicit — a service author has to know the
  convention rather than reading it off a wrapper name like the old `BodyID(..., http.StatusNoContent)`
  did explicitly. `generic_test.go` covers it so a regression fails loudly.
- `Query[Q]`'s stringly-typed decode pushes int/enum/date parsing into each service (already true
  for `/jobs` before this ADR); it's one extra small function per query route, not a new pattern.

## Rollout

One PR: the ADR 0020 rollout was already in progress but incomplete, so this folds into
finishing it rather than a separate migration.
