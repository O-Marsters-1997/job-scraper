# ADR 0031 — List bounds and retention

Issue: [#874](https://github.com/O-Marsters-1997/job-scraper/issues/874). The postgres-best-practices skill
(pagination, retention) is the authority; implemented in W3 onward.

## Decision

- **Every list endpoint is capped.** No handler returns an unbounded set.
- **Keyset paging** is used for candidates, events and score feedback. The cursor encodes the sort key plus
  id.
- **Offset plus total** is used for lists shown as numbered tables: jobs and companies. `page_size` is clamped
  to 100 (`docs/jobs-server-paging.md`).
- **A scheduled job deletes in batches**: done and dead `effect_outbox` rows and run logs older than 30 days.
- **`events` is kept forever** (ADR 0026) and indexed for the queries that read it.

## Consequences

- The Relevant sort in ADR 0022 runs in SQL, over the offset-paged jobs list.
- Deleting by age needs an index on the age column of each pruned table.
