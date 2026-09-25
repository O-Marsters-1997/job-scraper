# ADR 0022 — `Handle` as the one pipeline every handler goes through

**Status:** Accepted

## Context

ADR 0021 gave `internal/handlers` six CRUD-shaped generics (`GetAll`, `GetByID`, `Query`,
`Create`, `Update`, `Delete`) and let everything else — cookie-setting, redirecting, streaming
and service-token routes — stay a hand-written `func X(svc) http.HandlerFunc`, each repeating
its own version of "read the session, decode, call the service, map the error, write the
response." That's fine for the six shaped routes, which all funnel through the same adapter
helpers (`caller`, `decodeBody`/`decodeQuery`, `writeJSON`, `writeError`), but the misfits
(`Login`/`Signup`/`Logout`/`Me`, `OAuthStart`/`OAuthCallback`, `ExportCV`, `Ingest`/
`IngestBatch`) only share those helpers by convention — nothing stops a new misfit from
skipping a step, and the six generics themselves duplicate the same four-line
session/decode/call/error shell six times.

## Decision

- **`Handle[Req, Res any](decode, call, respond) http.HandlerFunc`** (`internal/handlers/generic.go`)
  is the one function every handler in the package is built from. `decode(r) (Req, error)` only
  reads the request — it never writes to `http.ResponseWriter` — and its error, like `call`'s,
  goes to the same `writeError`. `respond(w, r, res)` can't fail; anything that can fail belongs
  in `decode` or `call`.
- **Every route either binds a CRUD generic or calls `Handle` directly.** `GetAll`, `GetByID`,
  `Query`, `Create`, `Update` and `Delete` are now `Handle` with a fixed decode/respond pair —
  their exported signatures are unchanged, so nothing calling them moves. A route that sets
  cookies, redirects, streams, or authenticates by service token calls `Handle` with its own
  decode/call/respond instead of writing a bespoke `http.HandlerFunc` body. There is no third
  option: no handler in this package skips `Handle`.
- **The session read moves into `decode`.** `caller(w, r) (userID, ok)`, which wrote its own 401,
  becomes `userID(r) (string, error)`, returning `apperr.Unauthorized` on a missing session —
  handled by `writeError` like any other decode error. A route with no session requirement
  (`Login`, `Ingest`) simply doesn't call it.
- **`decodeBody[T](r) (T, error)`** drops the `http.ResponseWriter` parameter it used to take
  and write to; every caller now gets the error back and decides (in practice, always passes it
  straight through to `Handle`, which routes it to `writeError`).
- **A route that needs both a body and the caller's ID** (previously anything using
  `Create`/`Update`) decodes into a small package-private struct — `userInput[In]{userID, in}`
  for the CRUD generics, `sessionUser`, `oauthConnect`, `exportRequest` for the misfits — rather
  than `Handle` growing a third or fourth type parameter for "and also the caller."
- **Request→response types get the same `dto` convention as everywhere else.** Login/Signup's
  request becomes `dto.LoginInput`/`dto.SignupInput` and their response `dto.AuthUserView`;
  `Me`'s becomes `dto.MeView`. The service call signatures (`authSvc.Login(ctx, username,
  password)`, etc.) don't change — the handler's `call` step spreads the dto into them.
- **Two small, deliberate behaviour changes**, both accepted as the cost of one pipeline instead
  of a carve-out:
  - `Me` now 401s if the session is somehow missing from the request context, instead of
    silently writing an empty `{"id":"","username":""}`. In production this is unreachable
    (`auth.Middleware` already 401s first), so this only tightens a defensive fallback.
  - `OAuthCallback` clears its signed `oauth_state` cookie only when the callback succeeds,
    not on every failure path. The cookie is single-use in practice (`/oauth/start` overwrites
    it) and still expires on its own after 10 minutes, so nothing is left exploitable — `respond`
    genuinely can't run on an error, and clearing the cookie is a response-writing side effect.
- **The batch size cap moves to the route.** `IngestBatch`'s `decode` no longer wraps the body
  in `http.MaxBytesReader` (that needs `w`, which `decode` doesn't have); `POST /ingest/batch` in
  `router.go` gets `middleware.RequestSize(2 << 20)` instead. Same cap, enforced one layer up.
- **No new enforcement mechanism.** This ADR and `AGENTS.md` state the rule; a handler that
  doesn't go through `Handle` is something a reviewer (human or agent) should catch, the same
  way ADR 0021's "no closures in `router.go`" rule is enforced today.

## Consequences

- `internal/handlers` has exactly one code path from request to response, regardless of which
  handler you're reading — `decode`, `call`, `respond`, in that order, every time.
- Adding a genuine misfit route no longer means deciding how much of the adapter convention to
  reimplement by hand; it means picking three functions.
- The six CRUD generics are now ~5 lines each instead of ~15, because the session-resolve /
  decode / call / error-map shell they used to repeat lives once in `Handle`.
- `Me`'s and `OAuthCallback`'s behaviour changes are covered by `TestRouterRoutes` (the /auth/me
  and cv-templates/OAuth-adjacent subtests) and `TestHandle`
  (`internal/handlers/generic_test.go`), not by dedicated new tests — neither had unit coverage
  before this ADR.

## Supersedes

Supersedes the "misfits stay in `internal/handlers`, as functions over a service, not a
struct" clause of ADR 0021 (and, by extension, ADR 0020's original adapter helpers) to the
extent that clause implied those routes didn't need to share `Handle`'s structure — they still
live in `internal/handlers` as `func(svc) http.HandlerFunc` closures, they just build that
closure from `Handle` now.
