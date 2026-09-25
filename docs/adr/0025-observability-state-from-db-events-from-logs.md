# ADR 0025 — Observability: state from the database, events from logs

**Status:** Accepted

## Context

We want dashboards and alerts in Grafana Cloud for the api, worker and RabbitMQ, running on one
Docker Compose host. It has to stay on the free tier: 10k active metric series, 50GB of logs and
14 days of retention. The failure that costs a User most is a silent stop, where scraping or
scoring stalls and nothing is logged. We also need queue depth and latency, API errors and
latency, and per-User scoring spend, because each User's own key pays for their scoring.

Most of the state worth alerting on is already in Postgres (`effect_outbox`,
`board_poll_state`, `source_targets.run_status`, `harvest_runs`). Services already write
failures as slog JSON on stdout. There is no metrics or tracing code.

## Decision

- **Shipping is Grafana Alloy's job.** An `alloy` Compose service (profile `observability`,
  `env` external label) tails container stdout into Loki, scrapes `/metrics` from api, worker and
  the RabbitMQ prometheus plugin, and remote-writes everything to Grafana Cloud. The Go
  binaries hold no Grafana credentials and import no vendor SDK.
- **State comes from the database.** `internal/telemetry` holds one `prometheus.Collector`. On
  each scrape it runs one aggregate query (outbox backlog and oldest pending, Overdue Boards,
  failing Boards, failed runs, harvest age) through a small interface declared in that package.
  Only the api registers it. After a restart the gauges come back correct, because they read
  stored state instead of counting in memory.
- **Events come from logs.** No counters or histograms appear in domain code. Request, task and
  scoring-call rates, error ratios, p95 latencies and spend come from LogQL over slog lines. Every
  such line carries a stable `event` attribute (`http.request`, `task.done`, `score.call`), and
  the names are consts in `internal/telemetry`. Queries match on `event`, never on `msg`. Loki
  labels are limited to `service`, `level` and `env`.
- **`/metrics` is internal.** api and worker each serve it on a second listener (`METRICS_ADDR`,
  default `:9091`). That port is not published by Compose, so only Alloy can reach it.
- **Dashboards and alert rules are files** under `ops/grafana/`, pushed with `just grafana-push`.
  Alerts go to Grafana's email contact point, which doesn't route through Resend.

Rejected: pushing OTLP from the OTel SDK in each binary (credentials in every process, more
dependencies, exporter shutdown wiring, all to support traces we deferred); Prometheus
histograms injected into the router and broker constructors (constructor churn, and metrics split
across two sources); package-level `promauto` vars (global state).

## Consequences

- A log line with an `event` attribute is now a contract. Renaming or dropping one breaks a
  dashboard or alert, so change `ops/grafana/` in the same PR.
- Alerts on event rates run Loki queries, which are slower than PromQL. That's acceptable at this
  volume. If Loki query cost or log sampling becomes a problem, move the hot paths to
  histograms.
- Traces are out of scope. Adding them later means carrying trace context in AMQP headers and
  the ingest call, plus an OTLP receiver in Alloy.
- Plan: `plans/grafana-observability.md`.
