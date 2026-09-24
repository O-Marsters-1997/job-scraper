# Plan: RabbitMQ Source Work Queues

> Source: `.claude/handoffs/job-scraper-feat-issue-181__rabbitmq-migration-plan.md`; [ADR 0019](../docs/adr/0019-rabbitmq-source-work-queues.md) is the settled queue decision.

## Technical design decisions

- **Routes and triggers:** Keep the existing Source Target create/rerun API routes and Tracked Company Board cron. Creation and explicit rerun publish a Source Target start; cron publishes due verified Board checks at each Tracked Company's existing frequency. RabbitMQ never schedules a run. `--no-scrape` suppresses new scheduled starts but still drains accepted work; `--scrape-now` requests active Board checks without shifting their cadence.
- **Topology:** `internal/queue` owns one durable quorum queue `source.<name>` for every name in the Source registry, a direct work exchange keyed by Source, one durable quorum queue `source.dead`, and a dead-letter exchange bound to it. Declare topology at process startup and fail startup on mismatched declarations or missing bindings. The single worker process starts exactly one consuming goroutine per Source queue, with manual acknowledgements and prefetch one. Use separate AMQP channels for publishing and consuming; serialize publisher use as required by the selected Go AMQP client. Reconnect, redeclare, and resubscribe after broker restart.
- **Broker policy:** Pin RabbitMQ 4.3 in Compose. Source queues use `x-queue-type=quorum`, persistent messages, priority 8 for listing/Board tasks and 1 for details, `delivery-limit=5`, `delayed-retry-type=failed`, `delayed-retry-min=10000` ms, `delayed-retry-max=300000` ms, `dead-letter-strategy=at-least-once`, `overflow=reject-publish`, and a valid DLX/routing key. Declare the shared DLQ durable and quorum. `basic.reject(requeue=true)` records a failed attempt in 4.3; `basic.nack` does not advance the delivery count. Returned quorum messages lose their priority position, so listing priority is best effort. A single broker has one quorum member and no host-failure tolerance. See [quorum queue semantics](https://www.rabbitmq.com/docs/quorum-queues), [priority behavior](https://www.rabbitmq.com/docs/priority), and [dead-letter behavior](https://www.rabbitmq.com/docs/dlx).
- **Task contract:** A versioned typed envelope carries `task_id`, `source`, `kind` (`listing_page`, `detail`, `board_check`), `target_id` and `run_id` when a Source Target initiated the work, plus kind-specific `page_cursor`/target snapshot, `url`/Candidate identity, or `board_id` and `manual` flag. Validate the Source against the registry and require the fields for its kind before publish and before dispatch. A missing or unknown Source is an error; there is no unsourced detail queue. Route by the envelope's Source, not by URL detection. Keep the envelope and RabbitMQ `x-death`/delivery headers in the DLQ; log `task_id`, Source, kind, target/run ID, attempt, and cause.
- **Publish/consume contract:** `Publish(ctx, task) error` succeeds only after a publisher confirm **and** no mandatory return. An unroutable mandatory message can receive a positive confirm, so check both. A timeout or lost confirm is an unknown outcome and may lead to a duplicate publish. `Consume` acknowledges only after DB/API effects and every child publication is confirmed; handler failure rejects with requeue and lets broker retry/delay/DLQ policy act. Shutdown cancels consumers without acknowledging unfinished deliveries. Canonical Job upsert and Candidate uniqueness absorb duplicate detail work. See [publisher confirms](https://www.rabbitmq.com/docs/confirms) and [publisher recovery](https://www.rabbitmq.com/docs/reliability).
- **Source pages:** Keep existing `Source.Iterate` where it is useful for nonqueued callers, but add a page-at-a-time capability for discovery Sources: given a Source Target and opaque cursor, fetch one page and return cards, optional next cursor, and a terminal marker. WIS uses page number; LinkedIn uses its offset; Indeed, RemoteOK, and Remotive currently have one-page completion. A page task performs Candidate save/relevance assessment, confirms any detail tasks, then confirms at most one next-page task before acknowledging. Preserve the known-URL frontier and current Source-specific pagination limits. This serial continuation chain marks discovery success at the terminal page; later detail outcomes do not change Scrape Run status. ATS Board tasks fetch complete Jobs and have no detail child.
- **Postgres state:** Add nullable `source_targets.run_id UUID` as a generation token for new runs. Creation/rerun sets a fresh ID and `queued` state in one DB transaction; `running`/`succeeded`/`failed` updates compare both target ID and run ID so stale tasks cannot overwrite a newer run. Reconcile `queued` starts by republishing the same run ID. A fresh `running` duplicate is harmless; after a recovery timeout, a stale `running` start restarts discovery from page one for that same run, allowing duplicate work. Keep `board_poll_state` and its existing Board claim lease as the authority for due/active Board work; add no Source phase table or run-wide task counters. Add a small `harvest_runs(harvester TEXT PRIMARY KEY, last_succeeded_at TIMESTAMPTZ)` table for the current Valkey harvest gate; reuse Source Target and Board timestamps for their existing purposes. Use sqlc queries and numbered migrations.
- **Candidate boundary:** `SaveCards` must return `Card.Source=target.Source`; HTML enqueue paths must set Source explicitly. The Candidate assessment path may mark `detail_state='pending'` only after the detail publish is confirmed, or must reconcile any pending claim left before publish. Prefer confirm-before-pending: a crash may produce a duplicate detail task, while it cannot strand a Candidate as pending with no task. `Reconsider` follows the same ordering.
- **Operator interface:** Keep `cmd/queue` as the CLI entry point, replacing Valkey commands with RabbitMQ DLQ `list`/`inspect`/`replay` and counts. Use the management API's requeue option for non-destructive inspection, and AMQP manual ack for replay. Replay republishes the original typed task with a new task ID, waits for confirm and mandatory routing, and only then acknowledges the DLQ delivery; failed replay leaves it in the DLQ. Surface Source/kind/run and `x-death` context. Alert on DLQ depth and stalled Source queues; do not silently drop poison messages.
- **Authentication:** API authentication and Source Target ownership checks remain on the existing HTTP routes. RabbitMQ credentials come from environment/Compose secrets, with management bound to localhost. The worker and API use their own broker connections; the CLI uses operator credentials. No new public HTTP route is required.

## Recovery and cutover invariants

| Window | Owner and expected outcome |
| --- | --- |
| Source Target DB commit before start publish/confirm | Postgres reconciler republishes that run ID; a lost confirm can duplicate the start. CAS run state discards stale generations. |
| Child publish before parent ack | Broker redelivers the parent after a crash; child tasks can duplicate. Candidate and canonical Job identity absorb them. |
| Detail fetch/ingest before ack | Broker redelivers; API/DB canonical Job upsert remains idempotent. |
| Delayed listing retry while another run starts | Queue may interleave runs; run ID guards completion, and each Source still handles one delivery at a time. Existing details continue. |
| DLQ route unavailable | At-least-once dead lettering retains the exhausted message in its source quorum queue and retries transfer; `reject-publish` can backpressure new tasks. Alert on this condition. |
| Broker restart | Named volume restores confirmed messages; clients reconnect and redeclare. Unacknowledged work redelivers. Lost host volume requires backup restore and reconciliation, then manual replay where necessary. |

---

## Phase 1: Durable Source detail path

**User stories**: Accepted detail work survives process/broker restart; one Source executes one task at a time; failures retry and become inspectable.

### What to build

Add the RabbitMQ topology, typed task validation, publisher confirm/mandatory-return handling, and one sequential consumer per known Source. Route the existing Candidate and HTML detail producers through `Publish` using explicit Source identity, and retain the existing detail fetch → API exporter → canonical Job upsert path. Make the Candidate state transition safe across publish failures. Keep the old Valkey runtime available until the cutover phase, but run only one detail delivery path in a deployed environment.

### Acceptance criteria

- [ ] A known-Source detail task reaches the existing ingest path and is acknowledged only after success; missing/unknown Source and unroutable publish fail visibly.
- [ ] Two detail tasks for one Source execute sequentially; different Sources may progress concurrently.
- [ ] Real RabbitMQ integration check covers confirm plus mandatory return, broker restart/redelivery, `basic.reject` delayed retry and finite delivery limit to shared DLQ, and unavailable DLQ routing/backpressure. Keep this as one focused broker test suite.
- [ ] Candidate save/assessment and repeated detail delivery yield one canonical Job and no Candidate stuck pending without confirmed work.

---

## Phase 2: Resumable discovery pages

**User stories**: Discovery resumes after a worker crash; listing pages are preferred over details; a failed page does not stop already discovered details or later runs.

### What to build

Add `source_targets.run_id` and run-aware status updates, then give discovery Sources a one-page interface and turn each Target's Crawl into a chain of `listing_page` tasks. An internal start publisher can exercise this path with a persisted Target before the HTTP start path changes in Phase 3. A page applies the existing URL rewrite, Candidate save, relevance gate, and known-URL frontier; it publishes detail children and at most one next-page task, waits for all confirms, then acknowledges. Preserve WIS/LinkedIn pagination and one-page Sources. Mark the matching run succeeded on the terminal page, and show exhausted listing work as a failed run plus a DLQ entry. Later detail failures remain separate from run status.

### Acceptance criteria

- [ ] WIS and LinkedIn continue from a cursor after a process restart; one-page Sources finish without a second task.
- [ ] Page 1 can publish both details and page 2, and only acknowledges after both are confirmed; an uncertain child confirm can duplicate but cannot lose work.
- [ ] A retried page preserves Source on all Candidates and details, stops at the existing known-URL frontier, and cannot mark a newer run complete.
- [ ] Focused page-flow tests cover cursor progression, terminal success, a publish failure, and a delayed/terminal listing failure while an already-published detail still completes.

---

## Phase 3: Source Target starts and recovery

**User stories**: Creating or explicitly rerunning a Source Target reliably starts its Source work; a crash between DB state and publish cannot strand it.

### What to build

Wire the existing create and rerun HTTP handlers to the typed start task. Persist a new `run_id` with `queued` in Postgres, publish that ID, and use run-aware state transitions in the worker. On startup and a modest periodic sweep, republish queued and stale-running runs until a matching run completes or fails; throttle this sweep using DB timestamps so it does not flood the queue. Ignore duplicate starts for a fresh running or completed run; restart a stale running discovery from page one, tolerating duplicate cards and detail tasks. A newer rerun invalidates old tasks. Keep existing HTTP response semantics for publish errors, making the accepted DB run recoverable.

### Acceptance criteria

- [ ] Create and rerun reach the correct Source queue, including manual starts, without a general scrape-request queue.
- [ ] A forced failure after `queued` commit and before publish confirmation is repaired by reconciliation; a lost confirm only duplicates a harmless start.
- [ ] Stale run IDs cannot change the current Target's status; a successful discovery run ends when its last page finishes, regardless of detail tasks.
- [ ] Handler plus DB integration checks cover the DB/publish crash window, duplicate start, and newer rerun superseding an older one.

---

## Phase 4: ATS Board checks through Source queues

**User stories**: Due and manual verified Board checks share the Source's sequential worker; full ATS Jobs are ingested without detail tasks.

### What to build

Change cron and manual ATS starts to publish `board_check` tasks. The worker uses the existing `BoardPoller` fetch/ingest/complete behavior and `board_poll_state` claim, snapshot version, and lease; it does not add a queue lease. The cron keeps selecting due verified Boards using Tracked Company frequency. Due-Board sweeps after startup or publish uncertainty can republish; the Board claim/freshness check makes duplicate or stale scheduled tasks harmless. Manual Target starts carry target/run ID and finish that run after the Board check.

### Acceptance criteria

- [ ] Due Board task fetches and ingests complete Jobs, updates the Board snapshot, and creates no detail task.
- [ ] Manual ATS Target start uses the same Source queue and run-aware status, without shifting scheduled Board cadence.
- [ ] Repeated due-task publication, a lost confirm, and an expired Board claim cannot create an incorrect snapshot or duplicate canonical Jobs.
- [ ] One Board integration check covers due selection, claim contention, worker crash/retry, and manual completion.

---

## Phase 5: Operations and Valkey cutover

**User stories**: Operators can inspect and replay failures; scrape/harvest cadence survives Valkey removal; deployment and rollback are explicit.

### What to build

Move the remaining Valkey `scrape:last`/harvest gate to Postgres and wire the shared DLQ CLI and alerts. Replace Valkey with a pinned RabbitMQ 4.3 Compose service, named volume, credentials, health check, localhost-only management port, and documented volume backup/restore. Update `cmd/api`, `cmd/worker`, `cmd/queue`, `justfile`, README, environment examples, and tests; remove Valkey queue code, service, and Go/testcontainer dependencies. Perform a fresh cutover because there is no real Valkey work to import: stop publishers/workers, deploy DB migration and broker topology, verify binding/policies and a synthetic task, then switch API and worker together. Retain the previous image/config for rollback; if rolling back after RabbitMQ accepts real tasks, first drain or explicitly export/replay those tasks so rollback does not silently abandon them.

### Acceptance criteria

- [ ] No runtime Valkey references or dependencies remain; `just build`, `just test`, and `docker compose` startup checks pass.
- [ ] Harvester cadence persists through worker restart; Board and Source Target cadence still use their existing Postgres state.
- [ ] CLI lists/inspects the shared DLQ without consuming tasks and replay confirms publication before removing a DLQ entry; failed replay leaves the entry available.
- [ ] Compose restart preserves confirmed tasks; backup/restore instructions and the single-host volume-loss limitation are documented.
- [ ] Cutover checklist verifies Source queue count, DLX binding, retry policy, a successful detail task, a failed task reaching DLQ, and a recovered pending start before Valkey is removed.
