# ADR 0006 — RabbitMQ queues for source work

Accepted work must never be silently lost. A transient failure retries a bounded number of times, then goes to a dead-letter queue for inspection and replay. RabbitMQ provides this (durable queues, acks, delivery limits, dead-lettering) without hand-written scripts.

- One quorum queue and one sequential consumer per Source. Tasks carry an explicit kind (listing, detail, ATS board) plus the URL or board token and the Target/run identity. A task with no Source is rejected.
- Listing tasks publish their detail tasks and the next listing task, and confirm those publishes before acking. Listings have priority over details on a best-effort basis; redelivery can reorder them.
- Delivery is at least once, so processing must be idempotent. Duplicate tasks are fine because Postgres upserts keep one canonical Job.
- Cron and the API publish start tasks. Postgres, not the broker, owns cadence and timestamps, and reconciles starts lost before a publish confirm.
- It is deployed as one RabbitMQ container with a persistent volume. Losing that host's disk is a backup-restore event.
- A task with `Urgent` set routes to a shared `urgent` quorum queue instead of its source queue, with its own consumer, so a user-initiated task (a manual board verify-then-poll) never waits behind a source's backlog. Quorum queues have only two priority lanes and board checks already use the high one, so priority alone cannot jump the queue. The task keeps its Source, so per-source gates still apply. A deferral pauses the whole urgent consumer until the gate opens, so one gated source delays every source's urgent tasks; accepted because urgent tasks are rare and user-initiated.
