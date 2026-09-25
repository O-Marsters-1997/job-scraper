# ADR 0020 — Handlers as an HTTP adapter over feature services

**Status:** Accepted; implementation pending. Partially superseded by
[ADR 0022](0022-handle-as-the-one-handler-pipeline.md), which folds this ADR's adapter helpers
into a single `Handle` function every handler goes through.

## Context

Every handler in `internal/handlers` repeats the same steps inline: read the session, decode the body, validate, call one or more providers, map errors to status codes, and encode the response. The package has 41 session reads, 64 hand-written `"internal server error"` responses and 46 `Content-Type` headers. Business logic lives in handlers: `JobReasoningHandler` runs a full scoring use case, `SourceTargetHandler.Create` holds role guards and queue publishing, and `CompaniesHandler.Create`/`SetTracking` orchestrate several provider calls. Errors leak across modules: handlers check the Postgres `23505` code through `pgconn`, while `providers.ErrApplicationExists` goes unused. Error bodies come in three formats (plain text, JSON, and JSON sent as text/plain). Dependencies are injected four different ways, including concrete types and `With*` setters that mutate the handler. Adding an endpoint means copying this boilerplate, and testing a business rule means going through HTTP.

## Decision

- **Error kinds.** A new `internal/apperr` package defines kinds with fixed statuses: Invalid 400, Unauthorized 401, NotFound 404, Conflict 409, Unprocessable 422, Upstream 502, Unavailable 503. Sentinels are declared with a kind (`apperr.Conflict("board belongs to another company")`), including the existing provider sentinels. The db layer converts Postgres errors such as `23505` into provider sentinels, so nothing above the db layer imports `pgconn`.
- **Feature services.** Domain validation and orchestration move into per-feature packages (`internal/companies`, `internal/sourcetargets`, `internal/applications`, `internal/aiprefs`, `internal/scoringconfig`, `internal/jobreasoning`, alongside the existing `cvtemplates` and `candidates`). A service depends on `providers.X` interfaces for persistence and on small interfaces declared in its own package for anything else (queue publisher, board verifier, scorer, credential store). All dependencies are required constructor arguments: no `With*` setters and no nil checks.
- **Inputs and outputs.** Request bodies decode straight into `dto` input types, which carry the JSON tags. The caller's user ID and path IDs are never dto fields; they are passed as service arguments in the order `(ctx, userID, id…, in)`, so a request body cannot set them. Services validate every domain rule and return an `apperr` error when one fails. Services return wire-ready dto values; view shapes such as `AIPrefsView` are dto types.
- **The adapter.** `internal/handlers` becomes the HTTP adapter. It provides generic wrappers `User`, `ID`, `Body` and `BodyID` that take a function of the matching shape plus a success status, and it handles the session (401 if missing), decoding (400), kind → status mapping, turning a top-level nil slice into `[]`, and JSON encoding. Every error body is `{"error": msg}`. Only the adapter logs: errors without a kind are logged and returned as 500 with the message hidden. Services do not log errors they return.
- **Pass-through routes.** A route with no logic binds the adapter directly to a provider method value. A service method exists only when there is a rule or orchestration to hold. Provider signatures are reordered to `(ctx, userID, id…)` where needed so method values fit the adapter shapes.
- **Misfits.** Routes that set cookies (login, signup, logout), redirect (Google OAuth), stream (PDF export), authenticate by service token (ingest), are driven by query parameters (jobs paging, applications filters, board resolve) or need extra fields in the error body (the 409 with `count` when deleting a status in use) stay as plain `http.HandlerFunc`s. They use the shared helpers `Caller`, `DecodeJSON`, `WriteJSON` and `WriteError`, so they follow the same error contract, and their business logic still moves into services.
- **Wire format.** Apart from error bodies, no JSON field names or success statuses change.
- **Tests.** Business rules are tested on services with `providers.Mock*` and fake ports, without HTTP. The adapter is tested once for decoding, kind mapping, nil slices, the session and hiding 500 messages. One router-level table test checks that each route is wired to the right shape, path parameter and status. Most existing handler tests become service tests.
- **Rollout.** One PR.

## Consequences

- A new endpoint is a dto input type, a service method (or a provider method for a pass-through) and one line in `router.go`. It can be stubbed at the route by passing a closure and filled in from the service outward.
- An error has one definition, where it originates, and one HTTP mapping in the adapter; handlers no longer carry `errors.Is` chains.
- `dto` gains JSON tags on input types and couples to the wire format, including the existing camelCase and snake_case mix. This follows ADR 0001: `dto` owns data shapes passed between layers.
- Frontend code branches on status codes. Only `cvTemplates.ts` reads the error body, and it already expects JSON, so switching every error body to JSON breaks no client.
- Handler struct types mostly disappear; `router.go` wires services and providers into adapter calls.
- Doing it in a single PR avoids two coexisting styles, but produces a large diff that is harder to bisect if a route regresses. The route smoke test is the main guard.
