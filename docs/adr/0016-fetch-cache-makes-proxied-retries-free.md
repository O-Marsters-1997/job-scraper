# ADR 0016 — A fetch cache makes proxied retries free

> Amended by [ADR 0024](0024-hostile-sources-fetch-residential-only.md): Bright Data is gone, and the
> cache now wraps the Decodo residential route. Its Bright Data error-header and zone-probe details no
> longer apply; the rest stands.

A failed task is requeued and refetched from scratch (ADR 0006), so every retry of a LinkedIn or Indeed task paid Bright Data again. On 2026-09-30, with the API down, about 93 paid requests saved 4 jobs. Web Unlocker bills each response it delivers from the target, whatever its status, and bills none of its own failures (`x-brd-error` / `Proxy-Status`). The goal is that no retry of any post-fetch failure (export, parse, crash, DLQ replay) pays twice.

- **Where.** `fetchTransport.RoundTrip` in `internal/worker/proxy`, on proxied requests only, and only when the ctx carries a per-task collector. The processor attaches one per task, so the zone probe and non-task callers stay live.
- **What.** A 200 or a 3xx redirect hop without a Bright Data error header. Those are the responses a replay can use. `fetch_cache(url PK, status, header jsonb, body bytea, fetched_at)` stores the whole response, so `Get` and its redirect handling behave the same as on a live fetch. Other billed statuses are not cached: a 403 or 5xx may be transient, and replaying a cached one would stop a real retry from succeeding. There's no authwall detection: Unlocker turns detected protection pages into a `reject_block` 502, which is neither billed nor cached.
- **Gone.** A 404 or 410 means the job has gone, so retrying it only pays again. `Get` returns a typed gone error for either status, and a detail task that gets one logs it and acks without retrying. That costs one paid fetch per dead job.
- **Key.** The fetched URL, which is stable across redeliveries. `ReplayDead` issues a new task ID, so a key by task ID would miss on replay.
- **Owner.** `jobsearch`, which decides what to fetch (ADR 0011). The worker reaches it through facade methods (lookup, put, delete by URLs, delete expired) behind a local interface in `proxy`.
- **Lifetime.** Once the whole task has succeeded, the processor deletes the URLs the collector recorded, then acks. If that delete fails, the error is logged and the task is still acked. A daily sweep removes rows older than 7 days, which are left by tasks that dead-lettered and were never replayed. The rows are not deleted in the ingest transaction. A crash between the ingest commit and the ack would then pay on redelivery, ingest lives in the API, and ingest sees the exported URL, which LinkedIn rewrites to `applyUrl`, not the fetched one.
- **Failure.** A failed lookup fails the task without fetching, which costs a delivery attempt but no Bright Data spend. A failed write after a paid fetch is logged at ERROR and the task carries on. The worker stays up, and the existing Error burst and Dead letters alerts cover a database outage.

Trade-offs: retries become free but aren't prevented. With the API down, a task still dead-letters after 5 attempts, and its Job Candidate stays pending until it is replayed. Two duplicate detail tasks in flight at once can both miss the cache and both pay. A retried 403 or 5xx still pays each time. A gone job's Job Candidate stays pending until it expires.

Verified on 2026-09-30 by adding `-debug-full` to the zone username and reading `x-brd-debug`: a LinkedIn 200 and a LinkedIn target 404 both came back `billed=true`. An unresolvable host came back as a 502 with `x-brd-error` and `x-brd-error-code: proxy_error`, and was `billed=false`. The zone reports `used_req_headers=` as empty even when `Get`'s UA, Accept and Accept-Language headers are sent, so it isn't in the mode that bills every request.
