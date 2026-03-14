# job-scraper dev tasks

# list available recipes
default:
    @just --list

# ── Build ─────────────────────────────────────────────────────────────────────

# build the binary
build:
    go build -o bin/scraper ./cmd

# run the scraper
run *args:
    go run ./cmd {{args}}

# ── Code generation ───────────────────────────────────────────────────────────

# generate typed Go code from SQL files (requires sqlc: brew install sqlc)
generate:
    sqlc generate

# ── Code quality ──────────────────────────────────────────────────────────────

# format all Go files
fmt:
    gofmt -w .
    goimports -w .

# run linter
lint:
    golangci-lint run ./...

# run tests
test:
    go test ./...

# run tests with race detector
test-race:
    go test -race ./...

# ── Services ──────────────────────────────────────────────────────────────────

# start all services (postgres + valkey)
up:
    docker compose up -d
    @echo "services ready"

# stop all services
down:
    docker compose down

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
