# job-scraper

A Go service that discovers jobs from selected sources, deduplicates them in PostgreSQL, scores them for users, and optionally sends email notifications.

## How it runs

- `cmd/api` serves authenticated user routes and the service-token-protected `/ingest` endpoint. It owns canonical Job writes and the scoring outbox.
- `cmd/worker` publishes due verified ATS Board checks, reconciles unfinished Source Target starts, and consumes one durable RabbitMQ queue per known Source. It also harvests the company catalog and performs daily cleanup.
- `cmd/queue` lists, inspects, and replays tasks in the shared dead-letter queue.

Creating or rerunning a discovery Source Target stores a fresh run ID and `queued` status in PostgreSQL, then publishes its first listing-page task. WIS and LinkedIn continue through explicit page cursors; Indeed, RemoteOK, and Remotive finish in one page. A listing page saves and assesses Candidate cards, confirms detail and next-page tasks, then acknowledges. The matching Scrape Run succeeds at the terminal page; detail outcomes are tracked separately. Verified ATS Boards share the same Source queues and export complete Jobs without detail tasks. Scheduled checks keep each Tracked Company's frequency; manual checks do not shift it.

RabbitMQ 4.3 quorum queues use persistent messages, publisher confirms, manual acknowledgements, listing priority, delayed bounded retries, and one shared durable DLQ. The worker runs one sequential consumer per Source; different Sources can progress together. PostgreSQL run IDs fence stale work, and canonical Job identity absorbs duplicate deliveries. See [ADR 0019](docs/adr/0019-rabbitmq-source-work-queues.md) and the [operations guide](docs/operations/rabbitmq.md).

## Local setup

Requires Go 1.26, Docker, `just`, `goose`, and `sqlc` for query generation.

1. Copy `.env.example` to `.env` and set PostgreSQL, RabbitMQ, API, and ingest-token values. For Docker Compose, set `RABBITMQ_USER` and `RABBITMQ_PASSWORD` in the shell or `.env` before starting it. URL-encode special characters in `RABBITMQ_PASSWORD` when constructing `RABBITMQ_URL`.
2. Run `just up`, then `just migrate-up`.
3. Run `just build` or start the API and worker with `just run-api` and `just run`.

The worker needs `API_BASE_URL` and the same `INGEST_SERVICE_TOKEN` as the API. RabbitMQ management is exposed only on `127.0.0.1:15672` by Compose. The API and worker each make their own broker connection. `--no-scrape` suppresses new scheduled Board starts while draining accepted tasks; `--scrape-now` requests active verified Board checks without changing their cadence.

## Source Targets and Boards

Discovery searches start when created and can be rerun with `POST /source-targets/{id}/scrape`. Supported discovery Sources are `wis`, `linkedin`, `indeed`, `remoteok`, and `remotive`. ATS Sources are `greenhouse`, `lever`, `ashby`, `workable`, `recruitee`, and `personio`; they require a verified Board and are polled when due for a tracked Company. Source Targets, user ownership, and check frequencies remain in PostgreSQL. The authenticated API exposes `GET/POST /source-targets`, `PATCH/DELETE /source-targets/{id}`, and Company tracking routes.

An HTML Source implements `DetailFetcher`; ATS Sources return complete Jobs. Listing cards are saved as Candidates and assessed against the user's search config before detail publication. The worker fetches details using the task's Source identity and sends completed Jobs to `/ingest`. The API's canonical URL and provider identity rules deduplicate repeated deliveries.

To add a Source, see [Adding a Source](docs/sources/adding-a-source.md). A new Source must be registered before it can receive tasks; startup declares its queue and binding.

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

Run the migrations and bring up the pinned RabbitMQ container before switching the API and worker to this version. No production Valkey tasks need importing for the initial cutover. Follow the [RabbitMQ cutover, monitoring, backup, and rollback guide](docs/operations/rabbitmq.md) before removing the old deployment. Keep the broker's named volume and PostgreSQL backups together; one Compose host does not survive loss of its broker volume.
