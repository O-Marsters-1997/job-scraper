# ADR 0031 — List bounds and retention

Issue: [#874](https://github.com/O-Marsters-1997/job-scraper/issues/874). The postgres-best-practices skill
(pagination, retention) is the authority; implemented in W3 onward. Amends ADR 0022's "the list is already
loaded whole" assumption.

## Decision

- **Every list endpoint is capped.** No handler returns an unbounded set.
- **Keyset paging** replaces unbounded or offset lists for jobs, candidates, events, company pages and score
  feedback. The cursor encodes the sort key plus id.
- **A scheduled job deletes in batches**: done and dead `effect_outbox` rows and run logs older than 30 days.
- **`events` is kept forever** (ADR 0026) and indexed for the queries that read it.

## Consequences

- The Relevant sort in ADR 0022 runs client side over a fully loaded list; once jobs page by keyset, that sort
  has to move to the server. ADR 0022 already says so.
- Deleting by age needs an index on the age column of each pruned table.
