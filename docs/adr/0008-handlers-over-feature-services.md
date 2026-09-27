# ADR 0008 — Handlers as one `Handle` pipeline over feature services

- **Services** under `internal/api/services/<feature>` own validation and orchestration. They take required constructor deps (`providers.X` plus small interfaces declared locally) and return `apperr` errors and wire-ready `dto` values. Packages the worker also uses stay outside `services/`.
- **One pipeline.** Every handler in `internal/api/handlers` is built from `Handle[Req, Res](decode, call, respond)`. `decode` only reads the request (including the session, via `userID(r)`), `decode` and `call` errors both go to `writeError`, and `respond` can't fail. There is no third way to write a handler.
- **CRUD generics** (`GetAll`, `GetByID`, `Query`, `Create`, `Update`, `Delete`) are `Handle` with a fixed decode/respond and a fixed status; an `Out` of `struct{}` means 204. A route with no logic binds a provider method value; anything needing a closure becomes a service method.
- **Misfits** (cookies, redirects, streaming, service-token ingest) call `Handle` directly with their own decode/call/respond. A route needing both a body and the caller bundles them in a small package-private struct rather than `Handle` growing type parameters.
- **Inputs.** Bodies decode into `dto` types; path IDs fill `path:"…"`-tagged fields after decoding and the user ID is a service arg, so a body can never set either.
- **Errors.** `apperr` kinds map to fixed statuses; the db layer translates Postgres errors into sentinels. Every error body is `{"error": msg}`, with extra fields via `apperr.WithFields`. Only `writeError` logs, and an unkinded error becomes a 500 with the message hidden.
- **Tests.** Rules are tested on services with `providers.Mock*`, the adapter once in `generic_test.go`, and wiring with a router table test.

Trade-off: the `struct{}` → 204 convention and string-typed query decoding are implicit; `generic_test.go` guards them. Nothing mechanically enforces "every handler uses `Handle`"; review does.
