# ADR 0026 — Board verification runs in the worker

Confirming a Board (`POST /companies/{id}/boards` with `confirm: true`) publishes a `board_verify` task instead of fetching the Board inside the request. The worker fetches it once through the same adapters it polls with and marks the Board verified on success; a failure is logged and the Board stays a candidate, with no queue retry. The company page polls the Board list for up to 30 seconds and reports the outcome.

This keeps every outbound fetch, and with it `internal/sources` and `internal/proxy`, out of `cmd/api`. Source metadata the API does need (names, roles, filter fields) lives in `internal/sourcespec`, which imports no adapter.

Trade-off: the result arrives a few seconds later instead of in the response, and a worker that is down leaves the Board a candidate until it is retried.
