# ADR 0008 — Dead-letter and bounded retry for failed Enrich

**Status:** Accepted

## Context

The Enrich worker dequeues a URL with `ZPOPMIN` — which atomically *removes* it — then runs the handler. The handler error is logged but not acted on, so the URL is already gone from the queue. A transient failure (network blip, temporary block, soft 5xx) therefore permanently drops the Job; it only ever reappears by accident on a later Crawl that rediscovers the URL (and only because no DB row was written). In a scrape-once system this silent loss is a primary failure mode.

The scrape-once guarantee (`jobs.url UNIQUE` + `NewURLs` + queue `ZADD NX`) must not be weakened to fix this.

## Decision

On handler failure, **re-enqueue the URL with an incremented attempt counter**; once attempts reach a bounded cap **N**, move it to a **dead-letter sorted set** for inspection and manual replay rather than dropping it.

- Bounded retry absorbs transient failures.
- The dead-letter set makes terminal failures visible and recoverable instead of lost.
- Scrape-once is preserved: a URL is still detail-fetched at most once *successfully*; retries and dead-lettering operate on the same unique URL and never create duplicate Jobs.

Chosen over log-and-drop-after-N (the cheaper option), because the whole point is that a scrape-once system cannot rely on rediscovery to recover a lost Job.

## Consequences

- The queue gains attempt tracking and a second (dead-letter) sorted set. `MockQueue` does not replicate `ZADD NX`, so retry/dead-letter behaviour is covered by testcontainers-go integration tests against real Valkey, not the mock.
- Operators get a replayable dead-letter set; a draining-to-DLQ trend is a signal of a Source going hostile or a parser breaking.
- The cap `N` and backoff are configuration, not hardcoded.
