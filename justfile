# job-scraper dev tasks

set dotenv-load

MIGRATION_URL  := "postgres://" + env_var("POSTGRES_USER") + ":" + env_var("POSTGRES_PASSWORD") + "@localhost:5433/" + env_var("POSTGRES_DB") + "?sslmode=" + env_var_or_default("POSTGRES_SSLMODE", "disable")
MIGRATIONS_DIR := "scripts/migrations"

# list available recipes
default:
    @just --list

# ── Env ───────────────────────────────────────────────────────────────────────

# sync an env file's keys to .env.example: add missing keys with the example's value, drop removed ones
sync-env target=".env":
    ./scripts/sync-env.sh {{target}}

# ── Build ─────────────────────────────────────────────────────────────────────

# build all binaries
build:
    go build -o bin/api      ./cmd/api
    go build -o bin/worker   ./cmd/worker
    go build -o bin/admin    ./cmd/admin
    go build -o bin/snapshot ./cmd/snapshot
    go build -o bin/queue    ./cmd/queue

# run the worker
run *args:
    OTEL_SERVICE_NAME=worker METRICS_ADDR=:9092 LOG_FORMAT=text LOG_LEVEL=debug go run ./cmd/worker {{args}}

run-api:
    OTEL_SERVICE_NAME=api LOG_FORMAT=text LOG_LEVEL=debug go run ./cmd/api

# run a local Grafana + Tempo + OTLP collector (Grafana :3000, OTLP :4318, Tempo :3200); set OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
tracing-up:
    docker run --rm --name otel-lgtm -p 3000:3000 -p 4318:4318 -p 3200:3200 grafana/otel-lgtm

# print a fresh VAPID key pair for VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY
vapid-keys:
    go run ./cmd/vapidkeys

# ── Code generation ───────────────────────────────────────────────────────────

# generate typed Go code from SQL files (requires sqlc: brew install sqlc)
generate:
    sqlc generate

# build react-email templates to Go .tmpl files (requires bun)
build-emails:
    cd emails && bun run build.tsx

[doc('on the server: pull, build the frontend, install the Caddyfile and (re)start the stack')]
deploy:
    git pull --ff-only
    cd frontend && bun install --frozen-lockfile && VITE_API_URL=/api bun run build
    rsync -a --delete frontend/dist/ /opt/job-scraper/frontend/dist/
    sudo install -m 644 ops/caddy/Caddyfile /etc/caddy/Caddyfile
    sudo systemctl reload caddy
    docker compose up -d --build --remove-orphans
    docker compose ps

[doc('from the laptop: forward the app (localhost:8000) and RabbitMQ management (localhost:15673) from the server')]
tunnel host:
    ssh -N -L 8000:127.0.0.1:8000 -L 15673:127.0.0.1:15672 {{host}}

[doc('tail api and worker logs')]
logs:
    docker compose logs -f --tail=100 api worker

# push ops/grafana/ (contact point, notification policy, alert rules, dashboards) to Grafana
# (requires yq and jq; needs GRAFANA_URL and GRAFANA_SA_TOKEN)
grafana-push:
    ./scripts/grafana-push.sh

# ── Code quality ──────────────────────────────────────────────────────────────

# format all Go files (uses .golangci.yml formatters, same as CI)
fmt:
    golangci-lint fmt ./...

# run linter
lint:
    golangci-lint run ./...

# run tests
test:
    go test ./...

# run tailoring fixtures against real Sonnet (needs OPENROUTER_API_KEY; args: -runs N, -fixture NAME)
eval-tailoring *args:
    go run ./cmd/eval-tailoring {{args}}

# run tests with race detector
test-race:
    go test -race ./...

ci:
    just fmt
    just lint
    just generate
    just test
    just build

# ── Services ──────────────────────────────────────────────────────────────────

# start all services (postgres + RabbitMQ)
up:
    docker compose up -d
    @echo "services ready"

# stop all services
down:
    docker compose down

[doc("remove finished worktree stacks, their volumes and leftover test containers (--dry-run to preview)")]
clean *args:
    scripts/clean.sh {{args}}

# ── Migrations ────────────────────────────────────────────────────────────────

# show migration status
migrate-status:
    goose -dir {{MIGRATIONS_DIR}} postgres "{{MIGRATION_URL}}" status

# apply all pending migrations
migrate-up:
    goose -dir {{MIGRATIONS_DIR}} postgres "{{MIGRATION_URL}}" up

# rollback last migration
migrate-down:
    goose -dir {{MIGRATIONS_DIR}} postgres "{{MIGRATION_URL}}" down

# create a new named migration: just migrate-create add_index_on_url
migrate-create name:
    goose -dir {{MIGRATIONS_DIR}} create {{name}} sql

# ── Score Feedback ────────────────────────────────────────────────────────────

# print a user's Feedback Pack to stdout
scoring-feedback-export user *flags:
    go run ./cmd/admin scoring-feedback export {{user}} {{flags}}

# hard-delete a user's Score Feedback and print the count
scoring-feedback-clear user:
    go run ./cmd/admin scoring-feedback clear {{user}}

# replay a user's labels against current scoring and print the report
eval-scoring user:
    go run ./cmd/admin scoring replay {{user}}

# ── Seeds ─────────────────────────────────────────────────────────────────────

# seed the scoring-options bank (idempotent; safe to rerun in any environment)
seed-scoring-options:
    psql "{{MIGRATION_URL}}" -f scripts/seed/scoring_options.sql

# ── Database ──────────────────────────────────────────────────────────────────

# start postgres
db-up:
    docker compose up -d db
    docker compose exec db sh -c 'until pg_isready -U $POSTGRES_USER -d $POSTGRES_DB; do sleep 1; done'
    @echo "postgres is ready"

# stop postgres and remove data volume
db-reset:
    docker compose down -v

# tail postgres logs
db-logs:
    docker compose logs -f db

# open a psql shell
db-shell:
    docker compose exec db psql -U $POSTGRES_USER -d $POSTGRES_DB

# ── RabbitMQ ──────────────────────────────────────────────────────────────────

# show broker status
rabbitmq-status:
    docker compose exec rabbitmq rabbitmq-diagnostics status

# show dead-letter count
queue-list:
    docker compose exec worker ./queue count

queue-dead *args:
    docker compose exec worker ./queue {{args}}

[doc('purge every source queue and the dead-letter queue')]
rabbitmq-purge:
    for q in $(docker compose exec -T rabbitmq rabbitmqctl -q list_queues name | grep '^source\.'); do docker compose exec -T rabbitmq rabbitmqctl purge_queue "$q"; done
    docker compose exec -T rabbitmq rabbitmqctl -q list_queues name messages

# ── Snapshots ─────────────────────────────────────────────────────────────────

# run the snapshot CLI: just cli download wis page1 "https://..." | just cli rebase wis
cli *args:
    go run ./cmd/snapshot {{args}}
