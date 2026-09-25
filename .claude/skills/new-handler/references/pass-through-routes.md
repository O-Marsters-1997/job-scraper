# Pass-through routes

Per [ADR 0020](../../../../docs/adr/0020-handlers-as-http-adapter-over-feature-services.md): a route with
no domain logic binds a generic wrapper directly to a provider method value instead of routing
through a service. Add a service method only when there's an actual rule or orchestration to
hold — a service method that just forwards `(ctx, userID, ...)` to one provider call adds a
layer with nothing in it.

This is real and current:

```go
r.Get("/companies", handlers.GetAll(db.ListCompaniesForUser))
r.Get("/scores/status", handlers.GetAll(db.GetScoringStatus))
r.Get("/source-targets", handlers.GetAll(db.ListSourceTargetsByUser))
```

`db.ListCompaniesForUser` already has the shape `handlers.GetAll` wants —
`func(ctx, userID) (Out, error)` — so it binds straight to the route with no service in
between.

The moment the route needs anything past that — reordering arguments to match the wrapper,
converting the output type, validating input, calling more than one provider, deciding a
status by branching — write a named method on a service instead of a closure in `router.go`.
`companies.AddBoard` is the example: `AddCompanyBoardInput` carries the company ID (via its
`path:"id"` tag) and the URL to resolve, and the method validates, resolves the board and
calls two providers. There is no shortcut version of that logic that still belongs in
`router.go`.

If you're tempted to write `handlers.Create(func(ctx, userID string, in dto.X) (dto.Y, error) { ... })`
directly in `router.go`, that closure is the tell: give it a name and move it into the
feature's service package instead.
