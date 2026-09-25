# Pass-through routes

Per [ADR 0020](../../../../docs/adr/0020-handlers-as-http-adapter-over-feature-services.md):
a route with no domain logic binds the adapter directly to a provider method value instead of
routing through a service. Add a service method only when there's an actual rule or
orchestration to hold — a service that just forwards `(ctx, userID, in)` to one provider call
adds a layer with nothing in it.

This is part of the target ADR 0020 shape and isn't broadly present yet — `internal/router.go`
currently binds every route to a handler struct method, never a provider method value, because
provider method signatures don't match `http.HandlerFunc` and no adapter exists to bridge
them. When the adapter wrappers land, a genuine pass-through will look like:

```go
r.Get("/profile", handlers.User(profileProvider.GetProfile, http.StatusOK))
```

...instead of a `ProfileHandler` type with a hand-written method. Until then, keep writing the
handler struct + method like the rest of `internal/handlers` — see
`internal/handlers/profile.go` for the plainest example — and don't invent a one-off adapter
for a single route ahead of the rollout.
