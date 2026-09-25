# Routes that don't fit the adapter

Per [ADR 0020](../../../../docs/adr/0020-handlers-as-http-adapter-over-services.md), a few routes
never move onto `internal/handlers/generic.go`'s wrappers, because they need something the
wrappers don't support:

- **Set cookies** — login, signup, logout (`internal/handlers/auth.go`).
- **Redirect** — Google OAuth start/callback (`internal/handlers/google.go`).
- **Stream a response** — PDF export (`internal/handlers/cvtemplates.go`, `ExportCV`).
- **Authenticate by service token instead of session** — ingest
  (`internal/handlers/ingest.go`, mounted under `auth.ServiceTokenMiddleware` in
  `internal/router.go`).

These are written as `func X(svc) http.HandlerFunc` — a constructor that closes over the
service and returns the actual `http.HandlerFunc` — not a handler struct with methods, and not
a bare `http.HandlerFunc` with the service reached some other way. `internal/handlers/auth.go`
and `internal/handlers/google.go` are the templates.

They still use the shared, now-private, `caller`, `decodeBody`/`decodeQuery`, `writeJSON` and
`writeError` helpers from `adapter.go`/`generic.go` for the session, decoding and response
writing, so they follow the same `{"error": msg}` contract as every generic-wrapped route even
though they're not going through one.

The actual domain logic behind these routes — password hashing, session creation, OAuth token
exchange, the UserInfo lookup — lives in `internal/services/auth` and `internal/services/google`
like any other service; only the cookie/redirect/stream/service-token mechanics stay in the
handler. Before writing a new misfit route, check whether it's really one: `GET /google/status`
and `DELETE /google/link` look like OAuth routes but don't set cookies or redirect, so they
bind through `handlers.GetAll`/`handlers.Delete` like anything else — only `oauth/start` and
`oauth/callback` are genuine misfits in that file.

**Query-string-driven reads are not misfits any more.** `handlers.Query[Q]` (see the main
skill) covers jobs paging, applications filters and board resolve — decode the query into a dto
and bind through `Query` like any other read.
