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
