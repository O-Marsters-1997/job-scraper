# Pass-through routes

Per [ADR 0008](../../../../docs/adr/0008-handlers-over-feature-services.md), a route with no
domain logic binds a generic wrapper directly to a store method value in the module's `Routes`,
without going through a service. Add a service method only when there's an actual rule or
orchestration to hold. A service method that just forwards `(ctx, userID, ...)` to one store
call adds a layer with nothing in it.

```go
func (m *Module) Routes(r chi.Router) {
	r.Get("/companies", handlers.GetAll(m.store.ListCompaniesForUser))
	r.Get("/source-targets", handlers.GetAll(m.store.ListSourceTargetsByUser))
}
```

`m.store.ListCompaniesForUser` already has the shape `handlers.GetAll` wants,
`func(ctx, userID) (Out, error)`, so it binds straight to the route. The module can reach its
own `internal/store` because `Routes` lives in the context's root package.

As soon as the route needs anything more, write a named method on a service instead of a closure
in `Routes`. "Anything more" means:

- reordering arguments to match the wrapper
- converting the output type
- validating input
- calling more than one store method or another module's facade
- choosing a status by branching

`jobsearch.Service.AddCompanyBoard` is the example. `AddCompanyBoardInput` carries the company ID (via its
`path:"id"` tag) and the URL to resolve. The method validates the input, resolves the board and
makes two store calls. No shortcut version of that logic belongs in `Routes`.

If you're about to write
`handlers.Create(func(ctx, userID string, in dto.X) (dto.Y, error) { ... })` in `Routes`, the
closure is the sign. Give it a name and move it into the context's service.

Legacy contexts (see `AGENTS.md` § Migration status) do the same in `internal/api/router.go`,
binding `db.Method` values.
