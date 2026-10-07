# job-scraper

A Go service that discovers jobs from selected sources, deduplicates them in PostgreSQL, scores them for users, and optionally sends email notifications.

## How it runs

- `cmd/api` serves authenticated user routes and the service-token-protected `/ingest` endpoint. It owns canonical Job writes and the scoring outbox.
- `cmd/worker` publishes due verified ATS Board checks, reconciles unfinished Source Target starts, and consumes one durable RabbitMQ queue per known Source. It also harvests the company catalog and performs daily cleanup.
- `cmd/queue` lists, inspects, and replays tasks in the shared dead-letter queue.

The API is a modular monolith split into `jobsearch`, `scoring`, `applications`, `cvtemplates` and `identity`, each owning its tables ([ADR 0011](docs/adr/0011-modular-monolith-by-context.md)).

Creating or rerunning a discovery Source Target stores a fresh run ID and `queued` status in PostgreSQL, then publishes its first listing-page task. WIS and LinkedIn continue through explicit page cursors; Indeed, RemoteOK, and Remotive finish in one page. A listing page saves and assesses Candidate cards, confirms detail and next-page tasks, then acknowledges. The matching Scrape Run succeeds at the terminal page; detail outcomes are tracked separately. Verified ATS Boards share the same Source queues and export complete Jobs without detail tasks. Scheduled checks keep each Tracked Company's frequency; manual checks do not shift it.

RabbitMQ 4.3 quorum queues use persistent messages, publisher confirms, manual acknowledgements, listing priority, delayed bounded retries, and one shared durable DLQ. The worker runs one sequential consumer per Source; different Sources can progress together. PostgreSQL run IDs fence stale work, and canonical Job identity absorbs duplicate deliveries. See [ADR 0006](docs/adr/0006-rabbitmq-source-work-queues.md).

## Local setup

Requires Go 1.26, Docker, `just`, `goose`, and `sqlc` for query generation.

1. Copy `.env.example` to `.env` and set PostgreSQL, RabbitMQ, API, and ingest-token values. For Docker Compose, set `RABBITMQ_USER` and `RABBITMQ_PASSWORD` in the shell or `.env` before starting it. URL-encode special characters in `RABBITMQ_PASSWORD` when constructing `RABBITMQ_URL`.
2. Run `just up`; its `migrate` service applies migrations and seeds before the API and worker start. Run `just migrate-up` when running the binaries on the host instead.
3. Run `just build` or start the API and worker with `just run-api` and `just run`.

The worker needs `API_BASE_URL` and the same `INGEST_SERVICE_TOKEN` as the API. RabbitMQ management is exposed only on `127.0.0.1:15672` by Compose. The API and worker each make their own broker connection. `--scrape-now` requests active verified Board checks without changing their cadence.

## Source Targets and Boards

Discovery searches start when created and can be rerun with `POST /source-targets/{id}/scrape`. Supported discovery Sources are `wis`, `linkedin`, `indeed`, `remoteok`, and `remotive`. ATS Sources are `greenhouse`, `lever`, `ashby`, `workable`, `recruitee`, `personio`, `pinpoint`, `teamtailor`, `hibob`, and `smartrecruiters`; they require a verified Board and are polled when due for a tracked Company. Source Targets, user ownership, and check frequencies remain in PostgreSQL. The authenticated API exposes `GET/POST /source-targets`, `PATCH/DELETE /source-targets/{id}`, and Company tracking routes.

An HTML Source implements `DetailFetcher`; ATS Sources return complete Jobs. Listing cards are saved as Candidates and assessed against the user's search config before detail publication. The worker fetches details using the task's Source identity and sends completed Jobs to `/ingest`. The API's canonical URL and provider identity rules deduplicate repeated deliveries.

To add a Source, copy an existing one (`internal/worker/sources/greenhouse` for an ATS API, `internal/worker/sources/wis` for an HTML listing), then register it in `internal/sourcespec/sourcespec.go` and wire it in `internal/worker/sources/builder/build.go`.

## Commands

```sh
just build
just test
just test-race
just queue-list                  # shared DLQ count
just queue-dead list 20          # non-destructive inspection
just queue-dead inspect TASK_ID
just queue-dead replay TASK_ID   # confirm new publish before removing DLQ entry
```

Tests use real PostgreSQL and RabbitMQ containers for the persistence and queue paths, so Docker is required for `just test`. Parser snapshots remain under each HTML Source package; `just cli download <source> <name> <url>` captures one and `just cli rebase <source>` updates its expected JSON.

## Deployment

Production is a Hetzner server holding its own clone of this repo, running the Compose stack, with Caddy on the host serving the frontend and proxying `/api` to the API ([`ops/caddy/Caddyfile`](ops/caddy/Caddyfile)). Every port binds to `127.0.0.1` and the firewall admits only SSH, so the app is reached through an SSH tunnel at `http://localhost:8000`. Code reaches the server only through git; the server differs from a laptop only in its gitignored env files.

On the server, once:

1. Clone the repo and run `sudo scripts/provision.sh` to install Docker, Caddy, `just` and `bun`, and enable the firewall. Log out and back in to pick up the `docker` group.
2. Write production values into `.env` (start from `.env.example`; `API_BASE_URL` must be `http://api:8080`), then `ln -s .env .env.docker-compose` so Compose interpolation and container env read the same file. If Bright Data needs its CA certificate, copy it to `certs/brightdata.crt` (gitignored, mounted into the worker) and set `BRIGHTDATA_CA_CERT=/app/certs/brightdata.crt`.

On the server, each release: `just deploy`. It pulls, builds the frontend against `/api`, installs the Caddyfile and runs `docker compose up -d --build`; the `migrate` service applies migrations and seeds before the API and worker start. `just logs` tails them.

From a laptop: `just tunnel USER@HOST`, then open `http://localhost:8000`. RabbitMQ management is forwarded to `http://localhost:15672`.

Never change `AI_CREDENTIAL_ENC_KEY` or `GOOGLE_TOKEN_ENC_KEY` once users exist: stored credentials become undecryptable. Back up the `db_data` and `rabbitmq_data` volumes together; one Compose host does not survive loss of its broker volume. `docker compose down -v` deletes both.
