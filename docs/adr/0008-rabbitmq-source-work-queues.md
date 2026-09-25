# ADR 0008 — RabbitMQ queues for source work

Accepted work must never be silently lost. A transient failure retries a bounded number of times, then goes to a dead-letter queue for inspection and replay. RabbitMQ provides this (durable queues, acks, delivery limits, dead-lettering) instead of hand-written Valkey scripts, which were removed.

- One quorum queue and one sequential consumer per Source. Tasks carry an explicit kind (listing, detail, ATS board) plus the URL or board token and the Target/run identity. A task with no Source is rejected.
- Listing tasks publish their detail tasks and the next listing task, and confirm those publishes before acking. Listings have priority over details on a best-effort basis; redelivery can reorder them.
- Delivery is at least once, so processing must be idempotent. Duplicate tasks are fine because Postgres upserts keep one canonical Job.
- Cron and the API publish start tasks. Postgres, not the broker, owns cadence and timestamps, and reconciles starts lost before a publish confirm.
- It is deployed as one RabbitMQ container with a persistent volume. Losing that host's disk is a backup-restore event.
