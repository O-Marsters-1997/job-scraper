# Architecture Review — Scalable multi-source job scraping

_Date: 2026-09-23. Read-only code assessment of the working tree based on `8946bd5`, including the pre-existing orchestrator edit. Diagram and overview documentation amended separately. Six lenses reviewed; no runtime, load, or paid-provider tests performed._

_Planning target agreed after review: one or a few Users tracking up to hundreds of Companies over the next year. Design choices below should be implemented proportionately to that scale._

_Collection cadence agreed after review: periodic Company-catalog harvests and selected ATS Board checks continue; job-board discovery searches run once per request or explicit rerun. The earlier tentative 14-day HTML detail refresh was superseded and is not planned._

## Executive summary

**A sound foundation for a selective, cost-conscious scraper; not yet ready for safe horizontal worker scaling.** Keep the API/worker/PostgreSQL/Valkey shape. Shared discovery, selected company targets, reusable ATS adapters and a common ingest boundary fit the product well. More services would not solve the present constraints.

The biggest gap is that cheap ATS polling still causes expensive repeated processing: all returned jobs are upserted, synchronously scored for tracking users, and offered to notification handling. Queue items are removed before success, and schedulers have no atomic board claims. Replication would amplify duplicate collection and downstream spending without reliable recovery.

“Scrape once” should mean **reuse expensive details per meaningful content version**, while periodically checking selected boards for new postings and availability. Stable identity, reliable work ownership, and idempotent downstream effects make that economic model work. Bright Data improves access to protected sources; it does not supply these application guarantees.

## System map

- **Entry points:** browser SPA → session-authenticated Go API; hourly worker scheduling; queued manual scrape requests; company harvest/crawl loops.
- **Collection:** shared company catalog records ATS mappings. Tracking creates enabled per-user targets. Due-target SQL gates polling; the builder deduplicates the same ATS board across users within a run. Discovery alone does not activate board polling.
- **Data flow:** HTML listings → URL rewrite/known-URL filter → Valkey detail queue → fetch → authenticated HTTP ingest. Full ATS/feed jobs go directly to ingest. API saves → records companies → scores matching users → submits optional emails synchronously.
- **State:** shared PostgreSQL holds jobs, catalog, targets, scores and user data; both API and worker access it. Valkey stores pending URLs, payloads, retry state, scrape requests and timestamps.
- **Dependencies:** direct public ATS/company sites; optional Bright Data Web Unlocker proxy; Anthropic, Google and Resend.
- **Runtime:** a modular application deployed as API and worker processes, with shared persistence. Every worker currently also starts schedulers and discovery loops.

Evidence: [worker wiring](../../cmd/worker/main.go), [orchestrator](../../internal/scraper/orchestrate.go), [builder](../../internal/sources/builder/build.go), [ingest](../../internal/ingest/ingest.go), [overview](../architecture/system-overview.md).

## Findings by dimension

### Simplicity & understandability

The main components are proportionate to the problem. `Source`, optional `DetailFetcher`, and shared `BoardSource` describe actual variation without an unnecessary plugin framework.

Polling policy is spread across [due-target SQL](../../internal/data/sqlc/queries/source_targets.sql), the builder, orchestrator, and `boardDone` in worker wiring. Manual scrapes deliberately bypass scheduled freshness/closure updates. Changing cadence or reconciliation requires understanding all four. Document one lifecycle and eventually encapsulate board completion behind a small operation.

Ingest comments describe nonblocking/fire-and-forget work, but calls are synchronous. Swallowing errors does not create asynchronous execution. The amended diagram and overview describe actual behavior and label future intent explicitly.

### Maintainability

Fixture tests and boundary tests for [board failures](../../internal/sources/board_test.go) and [registry/builder completeness](../../internal/sources/builder/build_test.go) are good foundations.

Board completion updates freshness before querying and marking closures; the [completion callback](../../internal/sources/board.go) returns no error. Persistence failure can therefore leave reconciliation incomplete while postponing the next check. Move completion into a testable operation with explicit success/completeness semantics; exercise failed persistence, replay, partial/empty inventories, closure and reopening.

Source knowledge is repeated in [registry](../../internal/sources/registry.go), [detection](../../internal/detect/detect.go), builder and provider specs. The [source guide](../sources/adding-a-source.md) still recommends older board iteration and proxy guidance. Consolidate stable metadata where useful and test detection → board-token extraction → adapter construction as one contract before expanding the source set.

### Extensibility

The adapter model is a strong fit for additional ATS providers. Keep `Source` as an escape hatch when pagination, authentication or incremental feeds outgrow `BoardSpec`.

**Identity is the more significant extension limit.** [Jobs](../../internal/data/sqlc/queries/jobs.sql) are unique by URL; [companies](../../scripts/migrations/20260702000000_create_companies.sql) use unique slugs and a single ATS pair. Mirrors, URL parameters, ATS migrations, company-name collisions and multiple boards complicate reuse. Prefer stable company identities, separate board associations, provider/board/posting IDs and URL aliases. Merge cross-source listings only with evidence of equivalence, not merely matching titles.

Detail fetchers are captured from boot-time sources in [worker wiring](../../cmd/worker/main.go), while scheduled sources reload. Enabling a supported HTML source later can enumerate URLs without installing its detail handler. Construct supported detail capabilities independently of due target instances, or refresh both consistently.

The existing [proxy transport](../../internal/proxy/proxy.go) is the intended Bright Data Web Unlocker integration. Preserve source parsing independently of direct/proxy routing, using explicit direct and Web Unlocker fetchers per source. ADR 0018 now requires credentials at worker startup and forbids direct fallback for protected sources; this is an agreed change to the current behavior. There is no planned managed Web Scraper API integration.

### Security

Central session middleware, separate service-token ingest authentication, and encrypted user credentials are useful boundaries. See [router](../../internal/router.go), [service-token authentication](../../internal/auth/service_token.go) and [credential store](../../internal/credstore/credstore.go).

Outbound fetching needs a stronger boundary. [Source URL validation](../../internal/handlers/source_targets.go) uses string-prefix matching, which does not establish a hostname boundary. [Careers crawling](../../internal/discover/crawl/crawl.go) follows third-party URLs and default redirects without private-address restrictions. Validate parsed scheme/host, resolved addresses and every redirect; bound response sizes and restrict worker network reach. Apply admission policy to direct and vendor-mediated requests.

Proxy transport disables TLS verification and silently permits direct fallback when credentials are missing. Trust the required provider CA or use a verified HTTPS API. The agreed design requires worker startup to fail on missing proxy configuration and protected-source checks to fail on invalid credentials, without direct fallback (ADR 0018). Per-user scrape-now limits, pending-work deduplication and provider budgets are necessary when authenticated users can trigger paid fetching. Separate worker/API secrets and DB grants as deployment expands; Compose configuration alone does not establish public production exposure.

### Performance & scalability

**Unchanged ATS jobs repeatedly incur processing.** [ATS export](../../internal/scraper/orchestrate.go) passes all relevant jobs onward; [BulkExport](../../internal/scraper/api_exporter.go) sends individual HTTP requests serially. [Save](../../internal/data/db/job.go) returns every upserted job; [ingest scoring](../../internal/score/ingest.go) has no content-version guard. Work grows roughly with open jobs × subscribers × polls, instead of new/changed jobs and new/changed subscriber inputs. Separate last-seen inventory observations from content changes and enqueue scoring/notification effects only when their idempotency keys require them.

**The queue is not crash-safe.** [Dequeue](../../internal/queue/queue.go) removes work and deletes its payload before the handler finishes. `Nack` requeues only the URL; future retry scores are not checked before `ZPOPMIN`. Enqueue publishes the URL before writing its payload. The agreed delivery requirement is retry after a claim lease expires, with ingestion safe to repeat. Use atomic enqueue and claim/lease/ack, preserve payloads through retries, honor due times, and retain replayable dead letters. Configure and verify the chosen durability policy.

**Replicas duplicate scheduling.** Due targets are read rather than claimed; timestamps are written after work, and ticks can overlap. Deduplication in one builder invocation is not distributed ownership. At the agreed near-term scale, keep one active scheduler and prevent overlapping ticks. Introduce atomic leases per canonical Board before adding collection replicas; work-item leases are still needed now for crash recovery.

A single [detail worker](../../internal/worker/worker.go) sleeps 10–15 seconds per item: a theoretical ceiling of roughly 240–360 details/hour before network and ingest latency. ATS boards also run sequentially within a source. After reliability changes, add bounded concurrency with per-host/provider limits and isolate scoring from acquisition. [Job listing SQL](../../internal/data/sqlc/queries/jobs.sql) returns the entire catalog with descriptions; introduce pagination and explicit availability filtering as the catalog grows.

### Modularity (deep modules + coupling & cohesion)

`BoardSource` hides substantial fetch/parse/continuation behavior behind a small spec. The authenticated ingest endpoint and job DTO provide a useful normalization seam. Preserve them.

The worker entry point owns scheduling, catalog enrichment, closure reconciliation, queue consumers and session maintenance. Shared database access is a reasonable present trade-off, but write ownership is not exclusive: the worker closes jobs while API upserts reopen them. Concurrent or delayed snapshots can conflict. Give board reconciliation a transaction/version boundary and explicit ownership before distributing it further. Shared polling state currently lives on per-user target rows; when introducing leases, move board poll state to one shared record while subscriptions retain interest and requested cadence.

API ingestion also couples persistence to per-user AI latency and email delivery. Commit jobs plus an outbox atomically, acknowledge durable acceptance, and let independently retryable consumers perform scoring and notifications. These can remain modules/process roles in this repository; they do not require separate microservices.

## Cross-cutting themes

The six lenses converge on three issues: acquisition identity is weaker than the intended reuse guarantee; work completion is not a durable state transition; and collection is coupled to repeated per-user enrichment. These matter more than the number of source adapters.

There is already useful lifecycle handling: scheduled successful board snapshots close missing jobs, and upserts reopen them. Empty snapshots deliberately skip closure detection, leaving genuinely empty boards unresolved. Future pagination or parser regressions must not turn incomplete inventories into mass closures. Require confirmed complete snapshots, distinguish failure from empty success, and apply a grace/confirmation policy. Revalidate stale company-to-ATS mappings rather than treating detection as permanent.

## Prioritised recommendations

1. **Make reuse explicit before increasing polling volume.** Establish canonical acquisition keys and new/changed/unchanged ingest outcomes; store inventory observations separately. Version scoring by job content, user inputs and model/rubric; score cached jobs for new subscribers. Persist unique notification events. Validate that replaying an unchanged board produces no additional AI calls or alerts.
2. **Make accepted work recoverable before adding replicas.** Repair atomic queue publication, due-time handling, payload retention and claim/lease/ack semantics; add board scheduling ownership. Commit downstream outbox work with job changes. Exercise crashes after claim, after save and before acknowledgement, plus concurrent scheduling of one board.
3. **Complete the lifecycle contract.** Reconcile only complete snapshots, handle confirmed empty boards, protect against out-of-order results, and renew stale ATS mappings. Test closure/reopening and failure without false closure or premature freshness advancement.
4. **Control Bright Data Web Unlocker access.** Preserve direct ATS polling. Fix outbound URL/TLS policy and source/user/provider request admission before widening access; fail protected-source checks visibly rather than falling back to direct access. ADR 0018 chooses Bright Data's zone spending limit rather than an application spending cap: pause Web Unlocker sources on zone exhaustion and probe daily for recovery while direct sources continue.
5. **Scale from measurements.** Remove automatic repeat searches from discovery sources while retaining periodic catalog harvesting and selected ATS Board checks. Add bounded pools, actual batch ingest, paginated reads, and metrics for queue age, poll lateness, unique jobs per paid request, unchanged-hit ratio, source failures, AI calls and spend. Load-test the agreed target of one or a few Users and up to hundreds of tracked Companies before declaring capacity. No measured throughput was supplied, so this review does not claim a supported company/job count.

The design lands as a credible MVP with the right component boundaries and an evolutionary path to scale. Reliability and incremental processing are the next architectural investment; service proliferation is not warranted by the evidence.
