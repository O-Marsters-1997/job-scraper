# Routes that don't fit a CRUD generic

Per [ADR 0008](../../../../docs/adr/0008-handlers-over-feature-services.md), a few routes never
bind to one of `internal/handlers/generic.go`'s CRUD generics (`GetAll`, `GetByID`, `Query`,
`Create`, `Update`, `Delete`). They need something those fixed decode/respond pairs don't
support:

| Need | Routes | Context |
|---|---|---|
| Set cookies | login, signup, logout | `identity` |
| Redirect | Google OAuth start/callback | `identity` |
| Stream a response | PDF export (`ExportCV`) | `cvtemplates` |
| Authenticate by service token instead of session | ingest, mounted under `ServiceTokenMiddleware` | `jobsearch` |

These routes still go through `Handle(decode, call, respond)`, the same function the CRUD
generics are built from. They just call it directly with their own decode/call/respond instead
of a fixed pair.

Write each one as `func x(svc) http.HandlerFunc` in the owning context's `routes.go`: a
constructor that closes over the service and returns `Handle(...)`. Don't write a handler struct
with methods, and don't reach the service some other way. Until `identity` moves,
`internal/api/apihandlers/auth.go` and `google.go` are the templates.

When `decode` needs the session, it calls `handlers.UserID(r) (string, error)` itself. A miss
returns `apperr.Unauthorized`, which `writeError` handles like any other decode error. A route
with no session requirement (login, ingest) just doesn't call it.

Misfits use the exported helpers `handlers.DecodeBody`, `DecodeQuery` and `WriteJSON`, the same
ones the CRUD generics use. That keeps them on the same `{"error": msg}` contract as every other
route, even though they aren't bound to a fixed wrapper.

The domain logic behind these routes lives in the context's services like any other: password
hashing, session creation, OAuth token exchange, the UserInfo lookup. Only the
cookie/redirect/stream/service-token mechanics stay in the handler.

Before writing a new misfit route, check whether it really is one. `GET /google/status` and
`DELETE /google/link` look like OAuth routes, but they don't set cookies or redirect, so they
bind through `handlers.GetAll`/`handlers.Delete` like anything else.

Reads driven by the query string are not misfits. `handlers.Query[Q]` covers jobs paging,
application filters and board resolve.
