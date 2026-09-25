# Routes that don't fit the adapter

Per [ADR 0020](../../../../docs/adr/0020-handlers-as-http-adapter-over-feature-services.md),
some routes never move onto the `User`/`ID`/`Body`/`BodyID` adapter wrappers, even after the
rollout, because they need something the wrappers don't support:

- **Set cookies** — login, signup, logout (`internal/handlers/auth.go`).
- **Redirect** — Google OAuth (`internal/handlers/google.go`, `OAuthStart`/`OAuthCallback`).
- **Stream a response** — PDF export (`internal/handlers/cvtemplates.go`, `ExportCV`).
- **Authenticate by service token instead of session** — ingest
  (`internal/handlers/ingest.go`, mounted under `auth.ServiceTokenMiddleware` in
  `internal/router.go`).
- **Driven by query parameters** rather than a path ID or body — jobs paging, applications
  filters, board resolve.
- **Need extra fields in the error body** — e.g. the 409 with a `count` when deleting an
  application status still in use (`internal/handlers/application_statuses.go`).

These stay plain `http.HandlerFunc`s. The ADR says they use shared `Caller`, `DecodeJSON`,
`WriteJSON` and `WriteError` helpers for the session, decoding and response writing, so they
still follow the adapter's error contract (`{"error": msg}`) without going through it.

**As of this writing those four helpers don't exist yet** — checked, no `func Caller`,
`DecodeJSON`, `WriteJSON` or `WriteError` anywhere under `internal/`. Every route above is
still written with raw `net/http` today: `json.NewDecoder(r.Body).Decode(...)`,
`json.NewEncoder(w).Encode(...)`, `http.Error(w, msg, status)`. If you're adding a route that
belongs in this category before the ADR rollout lands, follow that same raw style — see
`internal/handlers/auth.go` for the shape. Once the helpers land, migrate the route to use
them in the same PR that adds the helpers, not as a one-off.
