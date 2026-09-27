# ADR 0009 — The worker fetches, the API persists

Every outbound fetch happens in the worker; persisting, scoring and notifying happen in the API. `depguard` in `.golangci.yml` enforces the import boundary. Under ADR 0011 the worker imports context roots (`internal/services/<ctx>`) for the state it owns, such as board poll state and source target runs, but never `internal/api` and never a module's `Routes`.

- **Ingest.** The worker scrapes and posts jobs to `POST /ingest` in `cmd/api`, authenticated with a shared bearer token (`INGEST_SERVICE_TOKEN`, never exposed to browsers). That gives one ingest path for every source and keeps scorer/notifier failures and credentials out of the scraper process.
- **Fetches the API needs** go through the queue. Confirming a Board (`POST /companies/{id}/boards` with `confirm: true`) publishes a `board_verify` task; the worker fetches it once through the same adapters it polls with and marks it verified on success. A failure is logged and the Board stays a candidate, with no queue retry. The company page polls for up to 30 seconds and reports the outcome.
- Source metadata the API needs (names, roles, filter fields) lives in `internal/sourcespec`, which imports no adapter.

Trade-offs: the worker needs `API_BASE_URL`, and a network partition that outlasts the exporter's bounded 5xx retries drops those jobs, which is acceptable at personal volume. Board verification answers a few seconds late, and a worker that is down leaves the Board a candidate until retried.
