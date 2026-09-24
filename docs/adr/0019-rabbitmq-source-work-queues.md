# ADR 0019 — RabbitMQ queues for source work

**Status:** Accepted; implementation pending
**Partially supersedes:** ADR 0008 (Valkey retry and dead-letter mechanics)

## Context

Valkey currently carries detail jobs, scrape requests, per-Source tasks, retry state, and scrape timestamps. The source queue adds a global scheduler, leases, cooldowns, pauses, and Lua coordination around a shared worker pool. Listing pages are still traversed inside `Source.Iterate`; the source worker handles only detail tasks. The user wants one sequential worker per Source, durable work, simple retry and dead-letter handling, and no silent loss of accepted tasks.

RabbitMQ and Valkey were considered. RabbitMQ adds a broker to operate but supplies durable queues, acknowledgements, bounded redelivery, and dead-letter routing without maintaining the current queue scripts. One physical priority queue per Source is preferred over separate listing and detail queues, accepting that returned messages can lose priority and runs can interleave.

## Decision

- Replace every Valkey work queue with RabbitMQ, including manual scrape requests and detail work. Each known Source has one physical queue and one consuming goroutine in the single worker binary. A task with no Source is rejected. Manual starts enter the relevant Source queue rather than a separate general crawler.
- Use explicit task kinds and carry the URL or Board token plus Target or run identity. Listing tasks publish discovered detail tasks and the next listing task. ATS Board tasks ingest their complete results without a detail phase. Confirm child publications before acknowledging the parent task; a crash may repeat a task, so processing must be idempotent.
- Cron continues to find due verified Boards according to Tracked Company check frequencies. Discovery Source Targets start when created or explicitly rerun. Cron and the API publish start tasks; RabbitMQ does not decide scrape cadence. Postgres retains scrape and harvest timestamps after Valkey is removed.
- Give listing tasks higher priority than details as a best-effort preference. A failed listing does not block detail work or later runs. Delayed retries can interleave tasks from different runs, while each Source still processes only one delivered task at a time. A Scrape Run's status describes its Crawl or ATS Board check; later detail failures are tracked separately.
- Use durable RabbitMQ 4.3 quorum queues, persistent messages, publisher confirms, mandatory routing checks, manual acknowledgements, and prefetch one. Use bounded broker-managed delayed retries for listing and detail failures. In AMQP 0.9.1, reject a failed delivery for requeue so it counts toward the quorum queue delivery limit; send exhausted tasks through a configured dead-letter exchange to one durable shared DLQ. Retain Source, task kind, and run identity in the message; use RabbitMQ dead-letter headers and task-keyed error logs for failure inspection and manual replay.
- Configure at-least-once dead-lettering with `dead-letter-strategy=at-least-once`, `overflow=reject-publish`, and a valid dead-letter exchange and binding. A publishing failure must be visible. Reconcile pending Source Target starts and due Boards from Postgres to recover a crash before RabbitMQ confirms the start task; consumers ignore duplicate or stale starts.
- The first deployment is one RabbitMQ container in Docker Compose with a persistent volume, credentials, health check, management access restricted to localhost, and backups. It survives process restarts but does not tolerate loss of that host's storage. Existing Valkey data can be discarded at cutover because it contains no real work yet.
- The required uniqueness outcome is one canonical Job. Repeated fetches and ingest requests are acceptable; Postgres upserts and existing Job identity rules prevent duplicate Job records.

## Consequences

- Source implementations must expose resumable listing-page work; a URL alone is not sufficient to identify every task or carry Board and Target context. The current HTML and immediate Candidate paths must preserve Source identity before enqueueing.
- The current Valkey queue, shared acquisition pool, queue leases, cooldown and pause state, legacy detail and scrape-request consumers, and Valkey service can be removed. Existing Board polling state remains responsible for Board freshness and concurrent Board claims; it is not a queue lease.
- Queue priority does not guarantee a full listing phase before details, particularly after redelivery. A dead-lettered listing makes discovery fail visibly while already discovered details may still finish. Operators need a way to inspect and replay the shared DLQ.
- Publisher confirms and consumer acknowledgements protect accepted work, but a single-host volume loss remains a backup recovery event. Duplicate tasks are possible after uncertain confirmations or crashes and must not create duplicate Jobs.
