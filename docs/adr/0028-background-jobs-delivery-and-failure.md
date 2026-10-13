# ADR 0028 — Background jobs: at-least-once delivery and how they fail

Issue: [#874](https://github.com/O-Marsters-1997/job-scraper/issues/874). Extends ADR 0006 (RabbitMQ
work queues) and ADR 0009 (worker fetches, API persists). The go-idiomatic skill is the authority for
the rules below.

## Decision

Status: only the panic rule is built (#910). The rest are rules W2 onward implements and reviews against.

- **Delivery is at-least-once.** Every handler, queue task or scoring effect, must be safe to run twice.
- **Failures are retryable by default.** A returned error requeues the task until the attempt cap, then it is
  dead-lettered.
- **`queue.Permanent(err)` goes straight to the DLQ.** For an error no retry can fix, such as a malformed
  payload. It is not built yet. Until it lands, a permanent failure burns the full attempt cap.
- **A panic is recovered, logged with its stack and dead-lettered.** Built in #910: `errPanicked` in
  `internal/queue/broker.go` covers queue handlers and scoring effects.
- **Write state before publishing.** A task is published only after the row it refers to is committed, so a
  consumer never sees a task whose state is missing.
- **Advance a cursor only after success.** Poll state, offsets and watermarks move after the work they cover
  has committed; a crash replays the work, never skips it.
- **Renew leases.** Long-running claims (outbox rows, harvest runs) extend their lease while they work, so a
  slow task is not claimed by a second worker.

## Consequences

- Idempotency is a property of each handler, enforced by `ON CONFLICT` upserts and by the fetch cache
  (ADR 0016), not by the broker.
- A poison message costs `max attempts` deliveries until `queue.Permanent` exists.
- Top-of-chain components own logging (ADR 0013 amendment): the broker logs the failure once. Once `errsite.Mark`
  lands (ADR 0013 amendment) the line carries the `err_site` of the first seam.
