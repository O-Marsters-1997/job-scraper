# job-scraper dev tasks

DB_URL         := "postgres://postgres:postgres@localhost:5433/job_scraper"
MIGRATIONS_DIR := "scripts/migrations"

# list available recipes
default:
    @just --list

# ── Build ─────────────────────────────────────────────────────────────────────

# build all binaries
build:
    go build -o bin/api      ./cmd/api
    go build -o bin/worker   ./cmd/worker
    go build -o bin/admin    ./cmd/admin
    go build -o bin/snapshot ./cmd/snapshot

# run the worker
run *args:
    go run ./cmd/worker {{args}}

# ── Code generation ───────────────────────────────────────────────────────────

# generate typed Go code from SQL files (requires sqlc: brew install sqlc)
generate:
    sqlc generate

# build react-email templates to Go .tmpl files (requires bun)
build-emails:
    cd emails && bun run build.tsx

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

# start all services (postgres + valkey)
up:
    docker compose up -d
    @echo "services ready"

# stop all services
down:
    docker compose down

# ── Migrations ────────────────────────────────────────────────────────────────

# show migration status
migrate-status:
    goose -dir {{MIGRATIONS_DIR}} postgres "{{DB_URL}}" status

# apply all pending migrations
migrate-up:
    goose -dir {{MIGRATIONS_DIR}} postgres "{{DB_URL}}" up

# rollback last migration
migrate-down:
    goose -dir {{MIGRATIONS_DIR}} postgres "{{DB_URL}}" down

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

# ── Valkey ────────────────────────────────────────────────────────────────────

# open a valkey-cli shell
valkey-shell:
    docker compose exec valkey valkey-cli

# list all URLs in the pending jobs sorted set (with scores)
queue-list:
    docker compose exec valkey valkey-cli ZRANGE jobs:pending 0 -1 WITHSCORES

# ── Snapshots ─────────────────────────────────────────────────────────────────

# run the snapshot CLI: just cli download wis page1 "https://..." | just cli rebase wis
cli *args:
    go run ./cmd/snapshot {{args}}
