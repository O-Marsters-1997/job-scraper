# Plan: Grafana Cloud observability

> Source: grilling session 2026-09-25 (roadmap card "Observability with Grafana Cloud"). Decision
> record: ADR-0025.

Ship logs and metrics from the api, worker and RabbitMQ to Grafana Cloud's free tier, with
dashboards and email alerts for silent stops, queue drain, API errors and latency, and per-User
scoring spend. Traces are out of scope.

## Technical design decisions

Durable decisions that apply across all phases. Honours **ADR-0025** (state from the database,
events from logs); **ADR-0020/0023** (handlers stay thin, so the access log is router middleware);
**ADR-0022** (scoring runs in `cmd/api`); **ADR-0024** (Jev is the scorer, so spend comes from
`SuitabilityResult.Cost`).

### Free tier budget
- Limits: 10k active series, 50GB logs, 14-day retention.
- Loki labels: `service`, `level`, `env` only. Metric labels never carry user, job, Board or URL
  IDs. Everything else is parsed from the JSON line at query time.

### Integration: Alloy
- A new `alloy` service in `docker-compose.yml` under `profiles: [observability]`, config in
  `ops/alloy/config.alloy`. Prod starts with `--profile observability`; `just up` does not.
- Logs: `loki.source.docker` over the Docker socket. `service` comes from the compose service
  name and `level` is promoted from the slog JSON.
- Metrics: scrape `api:9091/metrics` and `worker:9091/metrics` (`job` = service name). From
  phase 4, also scrape `rabbitmq:15692/metrics/detailed?family=queue_coarse_metrics`, because the
  default `/metrics` endpoint only has aggregates.
- The external label `env` (`prod` / `dev`) goes on every series and stream. Every alert rule
  filters on `env="prod"`.
- Credentials: `GRAFANA_CLOUD_PROM_URL`, `GRAFANA_CLOUD_PROM_USER`, `GRAFANA_CLOUD_LOKI_URL`,
  `GRAFANA_CLOUD_LOKI_USER` and `GRAFANA_CLOUD_TOKEN` go in `.env.docker-compose`, documented in
  `.env.example`. Only Alloy reads them. The Go binaries hold no Grafana credentials and import
  no vendor SDK.

### Module: `internal/telemetry`
A deep module: the binaries see a handful of functions; queries, metric names and event names
stay inside.
- `Serve(ctx context.Context, addr string, reg *prometheus.Registry) error` runs a dedicated
  `http.Server` serving `promhttp.HandlerFor(reg, …)` at `/metrics`. It shuts down when `ctx`
  ends.
- `NewStateCollector(store StateReader) prometheus.Collector`. `Collect` calls
  `store.OpsState(ctx)` with a 5s timeout and emits `jobscraper_state_up 1` plus the gauges. On
  error it emits only `jobscraper_state_up 0`, so missing data never looks like zeros.
- `type StateReader interface { OpsState(ctx context.Context) (dto.OpsState, error) }` is
  declared here. `*db.DB` implements it.
- Event name consts: `EventHTTPRequest = "http.request"`, `EventTaskDone = "task.done"`,
  `EventScoreCall = "score.call"`.
- `AccessLog(next http.Handler) http.Handler` is the chi-compatible middleware that emits
  `http.request`.
- Isolated tests: the collector against a fake `StateReader` (`prometheus/testutil`), and the
  middleware against a capturing `slog.Handler`.
- No package-level state. Each binary builds its own `prometheus.NewRegistry()` and registers
  `collectors.NewGoCollector()` and `collectors.NewProcessCollector(…)`. `METRICS_ADDR` defaults
  to `:9091` and is not published by compose.

### Key model: `dto.OpsState`
```go
type OpsState struct {
    OutboxPending             int
    OutboxOldestPendingAge    time.Duration
    OutboxFailed              int
    BoardsOverdue             int
    BoardsFailing             int
    SourceTargetsFailed       int
    HarvestAge                map[string]time.Duration // keyed by harvester
}
```
Filled by one sqlc query in `internal/data/sqlc/queries/ops_state.sql` (a single statement with
scalar subselects) plus the existing `harvest_runs` table. Fields are added phase by phase.

### Metrics (state gauges)
| Metric | Definition |
|---|---|
| `jobscraper_state_up` | 1 if the last `OpsState` read succeeded |
| `jobscraper_outbox_pending` | `effect_outbox` status in (pending, running) |
| `jobscraper_outbox_oldest_pending_seconds` | `now() - min(created_at)` of the above, or 0 |
| `jobscraper_outbox_failed` | `effect_outbox` status = failed |
| `jobscraper_boards_overdue` | active Verified Boards with `next_due_at < now() - 1h` and no live lease |
| `jobscraper_boards_failing` | `board_poll_state.consecutive_failures >= 3` |
| `jobscraper_source_targets_failed` | `source_targets.run_status = 'failed'` |
| `jobscraper_harvest_age_seconds{harvester}` | `now() - harvest_runs.last_succeeded_at` |

### Log event contract
One slog line per event, with `event` set from a `telemetry` const. Dashboards and alerts match
on `event`, never on `msg`. Renaming or dropping an attribute means updating `ops/grafana/` in
the same PR.

| `event` | Emitted at | Attributes |
|---|---|---|
| `http.request` | `telemetry.AccessLog`, the first `r.Use` in `NewRouter` | `method`, `route` (chi `RoutePattern()`, never the raw path), `status`, `duration_ms` |
| `task.done` | the queue consumer, after the task handler returns | `source`, `kind`, `outcome` (`ok`/`error`), `wait_ms`, `duration_ms`, `task_id`, `run_id` |
| `score.call` | the outbox worker, after a successful `Score` | `user_id`, `model`, `cost_usd`, `input_tokens` |

- `wait_ms` = consume time − `delivery.Timestamp`. Publishing sets
  `amqp.Publishing.Timestamp`. A zero timestamp (messages already queued at deploy) omits
  `wait_ms`.
- `score.call` lives in the outbox because the scorer doesn't know the user. The scorer's own
  "suitability scored via jev" line is unchanged.
- `http.request` skips `OPTIONS`.

### Grafana as code
- `ops/grafana/alerts/*.yaml` holds Grafana-managed alert rules, one email contact point and one
  notification policy (group by `alertname`, repeat every 4h), all in provisioning format.
- `ops/grafana/dashboards/*.json` holds dashboards built in the UI and exported: "Pipeline health"
  and "Traffic & cost".
- `just grafana-push` runs curl against the Grafana HTTP API with `GRAFANA_URL` and
  `GRAFANA_SA_TOKEN`. It upserts by UID and is idempotent.
- Alerts go to the built-in email contact point. They never route through Resend.

### Alerts (all filter `env="prod"`; thresholds are starting values, tune after two weeks)
| Alert | Condition | For | Phase |
|---|---|---|---|
| Target down | `up{job=~"api\|worker"} == 0` | 5m | 1 |
| Error burst | ERROR lines per `service` over 10m > 20 | 0m | 1 |
| State scrape failing | `jobscraper_state_up == 0` | 10m | 2 |
| Scoring stalled | `jobscraper_outbox_oldest_pending_seconds > 1800` | 5m | 2 |
| Scoring failed | `jobscraper_outbox_failed > 0` | 5m | 2 |
| Boards overdue | `jobscraper_boards_overdue > 0` | 1h | 3 |
| Board failing | `jobscraper_boards_failing > 0` | 15m | 3 |
| Harvest stale | `jobscraper_harvest_age_seconds > 172800` (2 × the 24h `harvestInterval`) | 15m | 3 |
| Dead letters | `source.dead` messages ready > 0 | 5m | 4 |
| Queue not draining | `source.*` messages ready > 0 and `deriv(…[30m]) >= 0` | 30m | 4 |
| Queue wait high | p95 `wait_ms` of `task.done` over 30m > 900000 | 0m | 5 |
| API 5xx | 5xx / all `http.request` over 10m > 5%, with ≥ 20 requests | 0m | 6 |
| API slow | p95 `duration_ms` of `http.request` over 15m > 2000 | 0m | 6 |
| User spend | `sum by (user_id)` of `score.call` `cost_usd` over 24h > 2 | 0m | 7 |

---

## Phase 1: Pipeline and "a service is down"

**User stories**: If the api or worker dies, I get an email. Logs and runtime metrics from prod are searchable in Grafana Cloud.

### What to build
The tracer bullet: telemetry leaves the host and an alert reaches my inbox.
- `internal/telemetry.Serve`. `cmd/api` and `cmd/worker` each build a registry with the Go and
  process collectors and run `Serve` on `METRICS_ADDR` alongside their main work.
- The `alloy` compose service in the `observability` profile, `ops/alloy/config.alloy` with log
  shipping plus the api and worker scrapes, and the env vars in `.env.example`.
- `ops/grafana/` with the contact point, notification policy, and the Target down and Error burst
  rules. Add `just grafana-push`.
- A skeleton "Pipeline health" dashboard: `up` per service, and a log panel by service and level.

### Acceptance criteria
- [ ] From inside the compose network, `curl worker:9091/metrics` and `curl api:9091/metrics` return Go and process metrics; port 9091 isn't reachable from the host.
- [ ] `Serve` stops when its context is cancelled (unit test).
- [ ] With `--profile observability`, prod logs and `up` series show in Grafana Cloud labelled `env="prod"`; plain `just up` starts no Alloy.
- [ ] Stopping the worker in prod fires Target down by email within ~6 minutes, and restarting it resolves the alert.
- [ ] Running `just grafana-push` twice leaves exactly one copy of each rule and dashboard.

---

## Phase 2: Scoring stall

**User stories**: If scoring stops draining or effects start failing, I get an email before I miss a digest.

### What to build
- `dto.OpsState` with the outbox fields, the `ops_state.sql` query, `just generate`, and
  `(*db.DB).OpsState`.
- `telemetry.StateReader` and `telemetry.NewStateCollector`, registered by `cmd/api` only.
- Rules: State scrape failing, Scoring stalled, Scoring failed. Outbox panels on "Pipeline health".

### Acceptance criteria
- [ ] Collector test with a fake `StateReader` (`testutil.CollectAndCompare`) covers the happy path, and the failure path emits only `jobscraper_state_up 0`.
- [ ] `OpsState` against the shared `testDB` returns correct pending, oldest-age and failed values for seeded `effect_outbox` rows.
- [ ] The worker's `/metrics` has no `jobscraper_*` series.
- [ ] A pending effect older than 30m in prod fires Scoring stalled.

---

## Phase 3: Scraping stall

**User stories**: If Boards stop being checked, a Board keeps failing, or a catalog harvest stops, I get an email.

### What to build
- Add `BoardsOverdue`, `BoardsFailing`, `SourceTargetsFailed` and `HarvestAge` to `OpsState` and
  the query, and the matching gauges to the collector.
- Rules: Boards overdue, Board failing, Harvest stale. Board and harvest panels on "Pipeline health".

### Acceptance criteria
- [ ] `OpsState` against `testDB` counts an active Verified Board with `next_due_at` 2h ago and no lease as overdue; a leased or retired one is not counted.
- [ ] A Board with `consecutive_failures = 3` is counted as failing.
- [ ] Every row in `harvest_runs` appears as `jobscraper_harvest_age_seconds{harvester=…}`.
- [ ] Running the worker with `-no-scrape` in a dev env with Alloy on leads to Boards overdue firing once `next_due_at` has passed.

---

## Phase 4: Queue depth and dead letters

**User stories**: I can see how deep each source queue is, and I get an email when tasks dead-letter or a queue stops draining.

### What to build
- An Alloy scrape of the RabbitMQ prometheus plugin's detailed per-queue endpoint.
  Enable the plugin in the compose `rabbitmq` service if the image doesn't already.
- Rules: Dead letters, Queue not draining. Per-queue ready/unacked panels on "Pipeline health".
- `docs/operations/rabbitmq.md` gains a short "Alerts" note pointing at the rules and the
  existing dead-letter inspection commands.

### Acceptance criteria
- [ ] `rabbitmq_detailed_queue_messages_ready{queue="source.dead"}` and one series per `source.<name>` queue show in Grafana Cloud.
- [ ] Publishing a task that fails 5 times in a dev env lands it in `source.dead` and fires Dead letters.
- [ ] A normal scheduled Board burst does not fire Queue not draining.

---

## Phase 5: Queue latency

**User stories**: I can see how long tasks wait in the queue and how long they take, per source and kind, and I get an email when waits grow.

### What to build
- Publishing stamps `amqp.Publishing.Timestamp`.
- The consumer times the handler and emits `task.done` (see the contract table). The existing
  "source task failed" error line stays.
- Rule: Queue wait high. Panels for wait and duration p95 by `source` and `kind`.

### Acceptance criteria
- [ ] Broker test (RabbitMQ testcontainer) asserts a published message carries a non-zero Timestamp.
- [ ] Consumer test asserts exactly one `task.done` per delivery, with `outcome`, `duration_ms` ≥ 0 and `wait_ms` ≥ 0, and `outcome="error"` when the handler fails.
- [ ] A delivery with a zero Timestamp logs `task.done` without `wait_ms`.

---

## Phase 6: API errors and latency

**User stories**: I can see API traffic, errors and latency per route, and I get an email on an error spike or slowdown.

### What to build
- `telemetry.AccessLog` registered as the first middleware in `NewRouter`, using
  `middleware.WrapResponseWriter` to capture status.
- Rules: API 5xx, API slow. The "Traffic & cost" dashboard with rate, 5xx ratio and p95 by route.

### Acceptance criteria
- [ ] Middleware test asserts one `http.request` line with `route="/jobs/{id}"` for a request to `/jobs/123`, and the correct `status` and `duration_ms`.
- [ ] `OPTIONS` requests are not logged.
- [ ] Router test: a request through `NewRouter` produces an `http.request` line, so the middleware is wired.

---

## Phase 7: Per-user spend

**User stories**: I can see scoring spend per day, total and per User, and I get an email when any User's daily spend runs away.

### What to build
- The outbox worker emits `score.call` after a successful `Score`, using `effect.UserID`,
  `effect.Model`, `result.Cost` and the input tokens.
- Rule: User spend. Spend panels on "Traffic & cost" (total per day, top Users by 24h cost).

### Acceptance criteria
- [ ] Outbox test with `providers.Mock*` and a fake scorer asserts one `score.call` line with `user_id`, `model` and `cost_usd` matching the fake's result.
- [ ] A failed `Score` emits no `score.call`.
- [ ] A literal `"event"` key appears only with a `telemetry.Event*` value (grep check in review).

---

## Out of scope
- Traces (trace context through AMQP headers and the ingest call; an OTLP receiver in Alloy).
- BrightData / Web Unlocker monitoring, which already has its own checks.
- Frontend (Vercel) observability.
