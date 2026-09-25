# Routes that don't fit a CRUD generic

Per [ADR 0020](../../../../docs/adr/0020-handlers-as-http-adapter-over-services.md) and
[ADR 0023](../../../../docs/adr/0023-handle-as-the-one-handler-pipeline.md), a few routes never
bind to one of `internal/api/handlers/generic.go`'s CRUD generics (`GetAll`, `GetByID`, `Query`,
`Create`, `Update`, `Delete`), because they need something those fixed decode/respond pairs
don't support:

- **Set cookies** — login, signup, logout (`internal/api/handlers/auth.go`).
- **Redirect** — Google OAuth start/callback (`internal/api/handlers/google.go`).
- **Stream a response** — PDF export (`internal/api/handlers/cvtemplates.go`, `ExportCV`).
- **Authenticate by service token instead of session** — ingest
  (`internal/api/handlers/ingest.go`, mounted under `auth.ServiceTokenMiddleware` in
  `internal/api/router.go`).

These still go through `Handle(decode, call, respond)` — the same function the CRUD generics are
built from — just called directly with their own decode/call/respond instead of one of the fixed
pairs. They're written as `func X(svc) http.HandlerFunc` — a constructor that closes over the
service and returns `Handle(...)` — not a handler struct with methods, and not a bare
`http.HandlerFunc` with the service reached some other way. `internal/api/handlers/auth.go` and
`internal/api/handlers/google.go` are the templates.

`decode` resolves the session by calling `userID(r) (string, error)` itself where the route needs
one (it returns an `apperr.Unauthorized` on a miss, handled by `writeError` like any other decode
error); a route with no session requirement (`Login`, `Ingest`) just doesn't call it. They use
the same `decodeBody`/`decodeQuery`, `writeJSON` and `writeError` helpers as the CRUD generics,
so they follow the same `{"error": msg}` contract as every other route even though they're not
bound to a fixed wrapper.

The actual domain logic behind these routes — password hashing, session creation, OAuth token
exchange, the UserInfo lookup — lives in `internal/api/services/auth` and `internal/api/services/google`
like any other service; only the cookie/redirect/stream/service-token mechanics stay in the
handler. Before writing a new misfit route, check whether it's really one: `GET /google/status`
and `DELETE /google/link` look like OAuth routes but don't set cookies or redirect, so they
bind through `handlers.GetAll`/`handlers.Delete` like anything else — only `oauth/start` and
`oauth/callback` are genuine misfits in that file.

**Query-string-driven reads are not misfits any more.** `handlers.Query[Q]` (see the main
skill) covers jobs paging, applications filters and board resolve — decode the query into a dto
and bind through `Query` like any other read.
