# job-scraper dev tasks

set dotenv-load

DB_URL         := "postgres://" + env_var("POSTGRES_USER") + ":" + env_var("POSTGRES_PASSWORD") + "@" + env_var("POSTGRES_HOST") + ":" + env_var("POSTGRES_PORT") + "/" + env_var("POSTGRES_DB") + "?sslmode=" + env_var_or_default("POSTGRES_SSLMODE", "disable")
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
    METRICS_ADDR=:9092 go run ./cmd/worker {{args}}

run-api:
    go run ./cmd/api

# ── Code generation ───────────────────────────────────────────────────────────

# generate typed Go code from SQL files (requires sqlc: brew install sqlc)
generate:
    sqlc generate

# build react-email templates to Go .tmpl files (requires bun)
build-emails:
    cd emails && bun run build.tsx

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

# ── Database ──────────────────────────────────────────────────────────────────

# start postgres
db-up:
    docker compose up -d db
    docker compose exec db sh -c 'until pg_isready -U $POSTGRES_USER -d $POSTGRES_DB; do sleep 1; done'
    @echo "postgres is ready"

# stop postgres
db-down:
    docker compose down

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

# ── Snapshots ─────────────────────────────────────────────────────────────────

# run the snapshot CLI: just cli download wis page1 "https://..." | just cli rebase wis
cli *args:
    go run ./cmd/snapshot {{args}}
