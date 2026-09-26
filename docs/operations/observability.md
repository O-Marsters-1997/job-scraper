# Grafana Cloud observability: operations

See `plans/grafana-observability.md` and ADR-0025 for the design. This covers the human setup
step the plan calls out, and the Phase 1 acceptance criteria that need a live Grafana Cloud
stack to verify.

## One-time setup

1. Create a [Grafana Cloud](https://grafana.com/auth/sign-up/create-user) free-tier account and
   stack.
2. From the stack's "Connections" page, connect Prometheus and Loki (this provisions the
   `grafanacloud-<stack>-prom` and `grafanacloud-<stack>-logs` datasources and gives you their
   remote-write URLs and usernames).
3. Create a service account (Administration → Service accounts) with the `Admin` role, and a
   token for it. This is `GRAFANA_SA_TOKEN`, used only by `just grafana-push`.
4. Fill in `.env.docker-compose`: `GRAFANA_CLOUD_PROM_URL`, `GRAFANA_CLOUD_PROM_USER`,
   `GRAFANA_CLOUD_LOKI_URL`, `GRAFANA_CLOUD_LOKI_USER`, `GRAFANA_CLOUD_TOKEN` (a Cloud Access
   Policy token with `metrics:write` and `logs:write`), `GRAFANA_URL` (your stack's base URL) and
   `GRAFANA_SA_TOKEN`.
5. Run `just grafana-push` to provision the contact point, notification policy, alert rules and
   dashboard. Safe to re-run; it upserts by UID.
6. Start the stack with the profile: `docker compose --profile observability up -d`.

## Manual verification (Phase 1 acceptance criteria)

Everything below needs the live Grafana Cloud stack from step 6 and can't be verified headless:

- [ ] Prod logs and `up` series show in Grafana Cloud, labelled `env="prod"`.
- [ ] Stopping the worker (`docker compose stop worker`) fires **Target down** by email to
      `ollyn.marsters@gmail.com` within ~6 minutes; `docker compose start worker` resolves it.
- [ ] Running `just grafana-push` twice leaves exactly one copy of each contact point, rule and
      dashboard (verified locally against a throwaway Grafana OSS container during development —
      re-check against the real Cloud stack once it exists).

Everything else — the metrics endpoint, `Serve`'s shutdown behaviour, the compose profile gating,
and the Alloy config's syntax — is covered by `go test ./internal/telemetry/...` and
`docker compose --profile observability config`.

## Manual verification (Phase 2 acceptance criteria)

Everything below needs the live Grafana Cloud stack and can't be verified headless:

- [ ] A pending effect older than 30m in prod fires **Scoring stalled** by email to
      `ollyn.marsters@gmail.com` within ~5 minutes of crossing the threshold.

Everything else — `OpsState`'s counts and ages, the collector's happy and failure paths, and the
worker's `/metrics` carrying no `jobscraper_*` series — is covered by `go test ./internal/data/db/...`
and `go test ./internal/telemetry/...`.
