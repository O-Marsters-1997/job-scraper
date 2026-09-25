# ADR 0020 — Handlers as an HTTP adapter over feature services

Handlers had repeated session, decode, error-mapping and encode boilerplate, and they held business logic. Now:

- **Services** under `internal/services/<feature>` own validation and orchestration. They take required constructor deps (`providers.X` plus small interfaces declared locally) and return `apperr` errors and wire-ready `dto` values. Packages the worker also uses (`candidates`, `ingest`) stay outside `services/`.
- **Errors**: `apperr` kinds map to fixed statuses, and the db layer translates Postgres errors into sentinels. Every error body is `{"error": msg}`, with optional extra fields via `apperr.WithFields`. Only the adapter logs, and unkinded errors become a 500 with the message hidden.
- **Adapter**: `internal/handlers` has generics keyed by function shape (`GetAll`, `GetByID`, `Query`, `Create`, `Update`, `Delete`), each with a fixed status. An `Out` of `struct{}` means 204. Path IDs fill `path:"…"`-tagged dto fields after the body is decoded, so a body can't override them.
- **Router**: every line is `r.<Method>(path, handlers.<Verb>(fn))`. A route with no logic binds a provider method value; anything needing a closure becomes a service method instead.
- **Misfits** (cookies, redirects, streaming, service-token ingest) are hand-written `func(svc) http.HandlerFunc`s that follow the same error contract.
- **Tests**: rules are tested on services with `providers.Mock*`, the adapter is tested once, and a router table test checks the wiring.

Trade-off: the `struct{}` → 204 convention and string-typed query decoding are implicit, and `generic_test.go` guards them.
