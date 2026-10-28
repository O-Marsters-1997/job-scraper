# job-scraper: Grafana queries

Run from the repo root. `W=.claude/perf/gcx-env.sh`. Output carries hint lines as extra JSON
documents, so filter with `jq 'select(.data)'`. Checked against real data on 2026-10-25 unless
marked otherwise. For syntax, load `loki` (LogQL), `promql` (PromQL) or `tempo` (TraceQL).

## Loki: per-route latency in production (the main source)

```bash
Q='{service="job-scraper-api-1", env="prod"} |= "http request" | json | method="GET"'

# worst daily p95 per route over 7 days
$W logs metrics -d grafanacloud-logs "quantile_over_time(0.95, $Q | unwrap duration_ms [1d]) by (route)" \
  --since 7d --step 1d -o json | jq -r 'select(.data) | [.data.result[] | {r: .metric.route,
  p: ([(.values // [])[][1]|tonumber]|max // 0|floor)}] | sort_by(-.p)[] | "\(.p)ms\t\(.r)"'

# request volume per route over 7 days
$W logs metrics -d grafanacloud-logs "sum by (route) (count_over_time($Q [1d]))" --since 7d --step 1d -o json \
  | jq -r 'select(.data) | [.data.result[] | {r: .metric.route, n: ([(.values // [])[][1]|tonumber]|add)}] | sort_by(-.n)[] | "\(.n)\t\(.r)"'

# one route's p50/p95 in a window (use the measurement window +/- a few minutes)
$W logs metrics -d grafanacloud-logs \
  "quantile_over_time(0.95, {service=\"job-scraper-api-1\", env=\"prod\"} |= \"http request\" | json | route=\"/companies\" | unwrap duration_ms [5m])" \
  --from 2026-10-25T12:00:00Z --to 2026-10-25T13:00:00Z --step 5m -o json

# the slowest individual requests for a route (raw lines, capped)
$W logs query -d grafanacloud-logs '{service="job-scraper-api-1", env="prod"} |= "http request" | json | route="/companies" | duration_ms > 1000' \
  --since 24h --limit 20 -o json

# errors and timeouts around a window
$W logs query -d grafanacloud-logs '{service="job-scraper-api-1", env="prod", level="ERROR"}' --since 1h --limit 50 -o json
```

`route` is the chi pattern (`/jobs/{id}`), and is empty for unmatched paths. The API is behind
Caddy's `/api` prefix, but logged routes have no prefix.

## Mimir: runtime context (no request histogram exists)

```bash
$W metrics query -d grafanacloud-prom 'rate(process_cpu_seconds_total{job="api"}[5m])' --since 1h --step 1m -o json
$W metrics query -d grafanacloud-prom 'process_resident_memory_bytes{job="api"}' --since 6h --step 5m -o json
$W metrics query -d grafanacloud-prom 'go_goroutines{job="api"}' --since 1h --step 1m -o json
$W metrics query -d grafanacloud-prom 'rate(go_gc_duration_seconds_sum{job="api"}[5m])' --since 1h --step 1m -o json
```

Each returns one series for `job="api"`. The full name list comes from
`metrics list-names -d grafanacloud-prom --match '{job="api"}'`.

## Traces: local only (Cloud Tempo receives nothing)

With `just tracing-up` running and the API started with `OTEL_EXPORTER_OTLP_ENDPOINT`:

```bash
curl -sG localhost:3200/api/search --data-urlencode 'q={resource.service.name="api" && duration>200ms}' --data-urlencode limit=20
curl -sG localhost:3200/api/search --data-urlencode 'q={resource.service.name="api" && span.url.path="/companies"}' --data-urlencode limit=20
curl -s localhost:3200/api/traces/<trace_id>
```

*To be verified on first local run:* the attribute that carries the path (`otelhttp` puts
`url.path` on server spans; it can't see chi's route pattern) and the `otelpgx` span names and
attributes used for counting DB spans per request (`db.query.text` / `db.statement`).

## Profiles

None sent. Report adding Pyroscope only if CPU-bound handler time turns up.
