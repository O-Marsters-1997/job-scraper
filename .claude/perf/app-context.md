# job-scraper: app context for app-performance-testing

No secrets here: variable **names** only. Last verified 2026-10-25.

## Stack

- Go 1.26, chi v5 router, pgx v5 + sqlc, Postgres 17, RabbitMQ. Worker is a separate binary.
- Routes: each context's `internal/services/<ctx>/routes.go` (`Routes(r chi.Router)`), mounted in
  `internal/api/router.go`. Middleware order: recoverer → `otelhttp` → RequestID →
  `logger.Middleware` → `telemetry.AccessLog` → CORS → requireJSON → session auth.
- Handlers are thin (`internal/handlers/generic.go`); logic is in `internal/services/<ctx>/`;
  SQL in `internal/services/<ctx>/store/queries/*.sql`.

## Environments

| | Local (laptop) | Production (VPS) |
|---|---|---|
| Detect | `.env` has `ALLOY_ENV=dev` | `ALLOY_ENV` is anything else, or missing |
| API base URL | `http://localhost:8080` (no `/api` prefix) | on the VPS: `http://127.0.0.1:8080`; through Caddy `http://127.0.0.1:8000/api`; from the laptop after `just tunnel app`: `http://localhost:8000/api` |
| Data | seeded fake data (below) | real |

Check with: `grep -E '^ALLOY_ENV=' .env | cut -d= -f2` (prints only that value, not secrets).

## Starting the app locally

1. `just up` (db on `localhost:5433`, RabbitMQ).
2. For traces: `just tracing-up` (Grafana `:3000`, OTLP `:4318`, Tempo `:3200`).
3. `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 just run-api` (`OTEL_SERVICE_NAME=api`, `:8080`).
   Stop the compose `api` first if it's running (`docker compose stop api`): it also binds `:8080`.
4. Production-like limits (the 2 vCPU / 4 GB VPS, split as api 1 CPU / 768 MB and db 1 CPU / 1.5 GB,
   leaving the rest for worker and RabbitMQ):
   `docker compose -f docker-compose.yml -f docker-compose.perf.yml up -d --build db api`.

## Auth

Session cookie `session_id` (sha256 of the token stored in `sessions`). The seeder writes a
ready session per `perf-*` user into the manifest; `measure_endpoints.py --manifest` sends the
power user's cookie for `record: heavy` and a typical user's for `record: typical`.
Production: the user copies their own `session_id` cookie into an env var, e.g.
`export PERF_COOKIE='session_id=…'`, and the run passes `--cookie-env PERF_COOKIE`. Never echo it.

## Seeder (local only)

- `just perf-seed --scale 1x|10x|<n> [--seed 42]` → `cmd/perfseed`.
- Refuses unless `ALLOY_ENV=dev`, the DB host is `localhost`/`127.0.0.1`/`::1`/`db`/a socket, and
  `--yes-wipe-dev-data` is passed (the just recipe passes it).
- Wipes: all jobs, companies, boards, scrape runs and **all** scores (`job_scores`,
  `job_rejections`) plus `perf-*` users. **Keeps** real users and every row they own, and the
  jobs/companies those rows point at (tracked companies, applications, CVs, notes, …).
- 1× ≈ 25k companies, 31k boards, 80k jobs (skewed: company #0 has thousands), 3k scrape runs;
  20 users: `perf-power-1` (1.5k tracked, 5k scores, 75k rejections), `perf-typical-1..3`,
  `perf-light-1..16`. 1× takes ~80 s. Users stay at 20 at any scale; everything else scales.
- Manifest: `.perf-reports/seed-manifest.json` (gitignored; holds dev passwords and sessions).
  `known_ids` keys: `user`, `company`, `job`, `application`, `task`, `scrape_run`, each with
  `typical` and `heavy`. Placeholders in `endpoints.json` use them: `{job.heavy}`.
- Docker Compose service name for the DB: `db`.

## Database access (read-only use)

- Local: `docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'`
  (credentials resolve inside the container, never on the command line).
- Production: same command on the VPS. Use plain `EXPLAIN` only. *To be verified on first VPS run.*
- No slow-query log is configured (`log_min_duration_statement` unset). Possible fix:
  set it in the compose `db` command.

## Grafana (Cloud, `https://<stack>.grafana.net`, Grafana 13)

- Call gcx through **`.claude/perf/gcx-env.sh`** (maps `GRAFANA_URL`/`GRAFANA_SA_TOKEN` from
  `.env`). `gcx login` fails its own `/health` pre-check against this stack, so the wrapper is
  the supported path, on the laptop and the VPS alike (the VPS just needs `gcx` installed).
- Datasource UIDs: metrics `grafanacloud-prom`, logs `grafanacloud-logs`, traces
  `grafanacloud-traces`, profiles `grafanacloud-profiles`.
- What actually arrives (checked 2026-10-25):
  - **Logs (Loki):** only `env="prod"`; `service="job-scraper-api-1"` (also `-worker-1`, `-db-1`,
    `-rabbitmq-1`, `-alloy-1`). JSON lines; the access log is `msg="http request"` with `method`,
    `route` (chi pattern), `status`, `duration_ms`, `trace_id`. **This is the only per-route
    latency in production.**
  - **Metrics (Mimir):** `job` = `api`, `worker`, `rabbitmq`. Go runtime (`go_*`), `process_*`,
    and `jobscraper_*` state gauges. **No request-duration histogram, no DB pool metrics.**
  - **Traces:** none in Cloud. Prod doesn't export spans (its `trace_id`s are UUIDs, the ADR 0014
    fallback). Local traces only, via `just tracing-up`: `otelhttp` root span per request,
    `otelpgx` span per query.
  - **Profiles:** none.
- Possible fixes worth reporting: an HTTP duration histogram by route; OTLP export from prod to
  Cloud Tempo; pgxpool stats as metrics.

## Production log sources

| Source | Where | Read-only command |
|---|---|---|
| API access log | Loki (from either machine) | see `grafana-queries.md` |
| API/worker stdout | VPS | `docker compose logs --since 1h --no-color api \| grep -m 500 '"http request"'` |
| Caddy access log | VPS | **none configured**: `ops/caddy/Caddyfile` has no `log` directive. Possible fix: add `log { output file /var/log/caddy/access.log; format json }`, which includes request `duration`. |
| Postgres slow queries | VPS | none configured (see above) |

The VPS-only rows are *to be verified on first VPS run*.

## Endpoints

Default list: `.claude/perf/endpoints.json` (DB-bound GETs × typical/heavy). Routes bound by
external calls (seeding can't reproduce them; investigate in production via Loki):
`/tailoring/**` (LLM), `/cv-templates/**` and `/tailoring/cvs/**` (Google Docs),
`/ai-usage` and `/google/status` (third-party polling).

## Known findings (baseline for comparison)

- Prod GET p95 (worst day of 7, Loki, 2026-10-25): `/companies` 3240 ms,
  `/tailoring/jobs/{jobId}/skill-suggestions` 1801 ms, `/cv-templates` 1598 ms,
  `/companies/tracked` 1371 ms, `/jobs/overview` 832 ms, `/jobs` 494 ms.
- `PageCompaniesForUser` (`internal/services/jobsearch/store/queries/companies.sql`) counts jobs
  × scores per company in a `LATERAL` for every company before paging, plus per-row board
  subqueries. Inferred from code, not yet measured.
- 17 FK columns have no leading index (e.g. `applications.job_id`, `job_views.job_id`,
  `job_grades.job_id`, `tracked_companies.company_id`, `company_favourites.company_id`,
  `board_job_observations.job_id`). Measured: deleting 73k jobs took 4 min 25 s from FK cascade
  checks. Query:
  `select conrelid::regclass, conkey from pg_constraint c where contype='f' and not exists (select 1 from pg_index i where i.indrelid=c.conrelid and i.indkey[0]=c.conkey[1])`.
