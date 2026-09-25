# Plan: Scalable production scraping

> Source: [domain context](../CONTEXT.md), [system overview and diagram](../docs/architecture/system-overview.md), [scalability review](../docs/architecture-reviews/scalable-multi-source-scraping.md), and accepted [ADR 0016](../docs/adr/0016-companies-and-company-level-tracking.md) / [ADR 0009](../docs/adr/0009-brightdata-web-unlocker.md). The approach document predates these decisions; the newer ADRs and context govern this plan.
>
> Target: one or a few Users tracking up to hundreds of Companies over the next year. Keep the Go API, worker, PostgreSQL, Valkey, and SolidJS SPA. One active scheduler is sufficient until measured capacity or deployment needs justify replicas.

## Current position and distance to target

The system already has six direct ATS adapters, discovery sources, a shared Company catalog, per-User search settings and scores, a relevance gate, the `POST /ingest` seam, a Valkey detail queue, and a usable Companies UI. Those are the foundation, not work to rebuild.

| Concern | Current code | Target and remaining work |
| --- | --- | --- |
| Discovery | Configured searches recur on the worker schedule; rejected cards are discarded. | Search once on request, retain Job Candidates for 60 days, and reconsider them when interest changes. Keep YC/Getro catalog harvests periodic. |
| Company choice | `companies` stores one ATS pair; ATS `source_targets` encode tracking and frequency; untracked careers sites are crawled. | Track Company once, even without a Board; associate multiple Verified Boards; inspect only tracked careers sites and retire superseded Boards safely. |
| Board lifecycle | Due target rows are read without claims; completion touches freshness before closure; empty results never close Jobs. | Shared Board state, complete ordered snapshots, one nonempty omission or two complete empty checks, and replay-safe closure/reopening. |
| Job reuse | URL is the unique Job key; each ATS poll exports and upserts all passing Jobs; scoring runs again. | Trusted posting identity, URL aliases, content fingerprint, separate inventory observations, and no downstream work for unchanged content. |
| Work delivery | Detail `ZPOPMIN` deletes payload before success; retry score is not checked; scrape requests use `LPOP`. | Atomic publication, due-time claim/lease/ack, payload retention, replayable dead letters, and idempotent ingest. |
| Personalisation | Ingest scores synchronously, then calls email with score `0`; no durable effect ledger. | Durable per-User/version scoring and first-discovery, thresholded alert effects; cached Jobs are scored for new trackers. |
| Paid fetching | Web Unlocker transport is optional, falls back to direct, and skips TLS verification. | Explicit fetchers, mandatory worker configuration, protected-source failure isolation, zone pause/probe, and outbound request controls. |
| Capacity | One detail consumer sleeps 10–15 seconds per item; ATS export is serial; `/jobs` returns the full catalog. | Bounded per-host work, batch ingest, paginated reads, measurable queue age, poll lag, spend, and load-test evidence. |

## Technical design decisions

### Data and identity

- **Companies and Boards:** retain `companies.id` as the stable shared Company key and `slug` as a display/search alias, not the sole identity rule. Add `company_boards(id, company_id, source, board_token, status, verification_method, verified_at, last_linked_at, retired_at, created_at)` with `UNIQUE(source, board_token)` and an index on `(company_id, status)`. `status` is `candidate`, `verified`, or `retired`; only verified Boards are polled. A careers-site link or explicit User confirmation plus a successful ATS fetch is required to verify. A detected URL alone creates a candidate. Record ambiguous slug collisions for review instead of silently merging Companies. Migrate existing `companies.ats_source/ats_token` associations and ATS targets before removing their authority.
- **User interest:** add `tracked_companies(user_id, company_id, enabled, check_interval_minutes, created_at, updated_at, PRIMARY KEY(user_id, company_id))`. The requested interval belongs here; direct ATS URLs resolve to a Company and Board and then use this relation. `source_targets` remains for discovery searches only. Backfill one Company choice from each enabled ATS target, preserving the User's requested frequency. If multiple old targets map to one Company, use the shortest interval and report the collision.
- **Shared Board state:** add `board_poll_state(board_id PRIMARY KEY, last_completed_at, last_started_at, last_snapshot_version, consecutive_complete_empty, consecutive_failures, lease_owner, lease_until, next_due_at)`. Due time is derived from the shortest enabled `tracked_companies.check_interval_minutes` for that Board's Company. A manual check does not advance scheduled cadence. A single scheduler runs initially; an atomic Board lease and fencing/version check must exist before a second scheduler is enabled.
- **Jobs and URLs:** add nullable `jobs.company_id`, `jobs.primary_board_id`, `jobs.provider_posting_id`, `jobs.content_fingerprint`, `jobs.content_changed_at`, and `jobs.first_discovered_at`; backfill from current source/slug/URL where evidence permits. Add `job_urls(job_id, normalized_url, source, first_seen_at, last_seen_at, PRIMARY KEY(normalized_url))`, with a trusted unique key on `(primary_board_id, provider_posting_id)` when both exist. Do not merge different ATS providers by title, Company, or approximate similarity. A URL without trusted posting ID remains a Provisional Job until evidence permits attachment. Keep the latest full details, not full historical copies.
- **Candidates and observations:** add `job_candidates(id, normalized_url UNIQUE, source, card_title, card_company, card_location, first_seen_at, last_seen_at, expires_at, detail_state)`, `candidate_discoveries(candidate_id, source_target_id, last_seen_at, PRIMARY KEY(candidate_id, source_target_id))`, and a bounded `candidate_assessments(candidate_id, user_id, search_config_version, relevance, evaluated_at)` relation. Add `board_job_observations(board_id, job_id, last_seen_at, last_snapshot_version, PRIMARY KEY(board_id, job_id))`. Candidates expire 60 days after latest explicit rediscovery. Board observations track availability independently of Job content. Index expiry, due Candidates, open Jobs by Board, and `jobs(company_id, closed_at)`.
- **Per-User effects:** version scoring inputs by incrementing `search_config.version` on relevant edits and recording model/rubric identity. Extend `job_scores` with `job_fingerprint`, `search_config_version`, `model_id`, and `assessed_at`. Add `effect_outbox(id, kind, aggregate_key, payload, available_at, lease_owner, lease_until, attempts, state, UNIQUE(kind, aggregate_key))` and `notification_events(user_id, job_id, event_type, state, provider_message_id, created_at, UNIQUE(user_id, job_id, event_type))`. Keys distinguish new/changed Job assessments and first-discovery alerts. A changed rubric marks existing assessments stale; an explicit User rescore enqueues bounded work.

### Boundaries and contracts

- **Collection:** keep `sources.Source` and optional `DetailFetcher`. Board adapters emit a `BoardSnapshot` containing `board_id`, observed Jobs, `complete`, `started_at`, and source cursor/version. A Board check succeeds only after all required pages parse and new/changed content is durably accepted by ingest. `BoardPoller` owns claim → fetch → reconcile → complete/fail as one testable operation; adapters own fetch and parse, not database freshness or closure. Isolated tests cover partial pages, empty snapshots, failed persistence, replay, and out-of-order completion. HTML Job closure uses only an explicit closed signal or confirmed 404/410 from its own page; transient fetch errors leave it open.
- **Ingest:** keep service-token `POST /ingest` for one Job and add `POST /ingest/batch` for bounded ATS batches. Both call one `IngestJobs` boundary returning per-item `new | changed | unchanged | rejected` plus canonical Job ID; repeated submissions are safe. Validation and identity resolution occur before the transactional Job update. The transaction also writes necessary outbox effects. `APIExporter` retries ambiguous transport failures with the same logical keys. Isolated tests cover alias resolution, unchanged replay, and transaction rollback.
- **Detail work:** a queue item has stable ID, normalized URL, card payload, due time, attempt count, lease owner/expiry, and state. The queue exposes `Publish`, `ClaimReady`, `Ack`, `Nack`, and `ReplayDeadLetter` for both detail and scrape-request types. Publication and state changes are atomic; expired claims are reclaimable. Successful ingest is the acknowledgement boundary. Use Valkey scripts or transactions behind one queue module and integration-test against real Valkey. Configure and verify Valkey persistence/backup policy before treating it as durable work storage.
- **Personalisation:** an outbox consumer selects interested Users by Company tracking for ATS Jobs or matching discovery interest for Candidates, then scores only missing `(job fingerprint, User, config version, model)` combinations. A second consumer attempts notification only after a qualifying Suitability result and only for first discovery. A provider timeout may cause repeat submission; persist a provider idempotency key if supported and expose uncertain sends for operator review. No later edit, reopening, Candidate promotion, or new subscription creates a new-Job alert.
- **Fetch boundary:** source metadata selects direct or Web Unlocker `Fetcher`; parser contracts remain shared. Worker startup validates `BRIGHTDATA_PROXY_URL` even in a direct-only configuration. The Web Unlocker fetcher never falls back to direct and uses a trusted provider CA or verified HTTPS access. Zone exhaustion pauses only protected sources; a daily probe resumes them. Admission checks validate scheme, host, resolved IP, redirects, timeouts, response size, and per-host/provider concurrency for direct and mediated requests. Isolated tests cover fallback prevention, zone pause, redirects, and budget/error classification.
- **Auth and routes:** browser routes remain session-authenticated; ingest remains service-token authenticated. Keep `GET /companies`, `POST /companies`, and `PUT /companies/{id}/tracking`, extending tracking payload with `enabled` and `check_interval_minutes`. Add `GET /companies/{id}/boards` and `POST /companies/{id}/boards` for visible verification/confirmation, and include tracking/Board status in Company responses. Keep `/source-targets` for discovery searches and reject new ATS-role targets after migration. Add `POST /source-targets/{id}/scrape` for explicit rerun and `POST /scores/rescore` for the current User's bounded on-demand rescore. `GET /jobs` gains `cursor`, `limit`, `availability`, and stable sort inputs, returning `items` and `next_cursor`; `GET /jobs/{id}` supplies full detail. The UI moves Company tracking and Board status to `/companies` and `/companies/$id`, discovery reruns to Searches, and paginated results to `/jobs` and the Company detail page.
- **Scheduling and operations:** keep one active scheduler until measurements show a need for replicas. It periodically harvests the Company catalog, checks due verified Boards, and revalidates tracked careers sites about monthly or sooner after repeated Board failures. Discovery searches have no recurring schedule. Metrics distinguish source check success/completeness, Board lag, Candidate retention, queue age/leases/dead letters, unchanged ingest ratio, AI calls/cost, Web Unlocker use/zone pause, and notification attempts. Use bounded concurrency and provider limits, then load-test the agreed target before increasing worker replicas.

### Migration and cutover rules

Add nullable columns and new tables first; backfill in bounded, idempotent batches; dual-read or translate legacy ATS targets during the transition; switch API and worker behavior together; then stop writing legacy ATS target fields. Do not drop old columns or uniqueness constraints until the backfill, ambiguous mappings, and UI cutover are checked. Keep a rollback path that can disable new scheduling and resume one-worker legacy collection without losing accepted detail work. Later migrations can remove compatibility columns after production evidence.

---

## Phase 1: One-shot discovery with retained Candidates

**User stories:** As a User, I can run a discovery search when I add or rerun a Source Target; changing my interest can surface earlier Candidates without paying for a new search or fetching every detail page. As an operator, I can keep Company catalog harvesting periodic without recurring job-board searches.

### What to build

Remove discovery-role targets from the scheduled `BuildSources` path; retain creation-time and explicit rerun requests. Persist listing-card metadata before the relevance gate, including rejected Candidates, with the 60-day retention window and source/target provenance. Reconsider Candidates in bounded batches when a discovery target or Search Config changes, and enqueue detail only for a newly interested User. Keep periodic YC/Getro harvesting. Show last run and rerun state in Searches; make request failure recoverable in Phase 5 without changing this user contract.

### Acceptance criteria

- [ ] A discovery search runs once on creation or explicit rerun, and a scheduled tick does not repeat it; YC/Getro still harvest on schedule.
- [ ] Rejected card metadata remains queryable for 60 days and a Search Config change re-evaluates it without refetching the discovery page.
- [ ] A Candidate that now passes the cheap gate creates at most one detail request; a Candidate still rejected does not incur a detail fetch.
- [ ] The Searches UI shows run status and allows an explicit rerun; tests cover creation, change-triggered reconsideration, expiry, and duplicate cards.

---

## Phase 2: Company tracking and verified multi-Board associations

**User stories:** As a User, I can track a Company before its ATS is known, set one Check Frequency, and automatically include later verified Boards. As a User, I can add a direct ATS Board URL and confirm its Company association. As an operator, I avoid inspecting untracked careers sites.

### What to build

Add `tracked_companies` and `company_boards`, migrate existing Company mappings and ATS targets, and make `PUT /companies/{id}/tracking` authoritative. A direct ATS URL resolves to a Company and candidate Board; a successful Board fetch plus careers link or explicit User confirmation makes it verified. Tracking triggers a bounded initial careers inspection; monthly and failure-triggered rechecks apply only to tracked Companies. The UI shows unknown/candidate/verified/retired Board state and the User's requested frequency. Scoring interest queries switch from ATS target matching to Company tracking.

### Acceptance criteria

- [ ] A Company without a Board can be tracked and appears tracked across reloads; a newly verified Board joins without another User action.
- [ ] Two verified Boards for one Company are polled at the shortest active User interval, once each per due check; untracked Companies are not crawled for Boards.
- [ ] A detected ATS URL alone cannot start scheduled polling; failed verification stays visible and retryable.
- [ ] Legacy ATS targets backfill without losing enabled state or frequency; ambiguous Company matches are reported for review.
- [ ] Company API and UI expose tracking and Board status; discovery Source Targets still work.

---

## Phase 3: Complete and ordered Board reconciliation

**User stories:** As a User, I see Jobs close only when a trustworthy Board check proves they disappeared, and reopen when advertised again. As an operator, a failed or partial check cannot advance freshness or retire a Board.

### What to build

Introduce `BoardSnapshot` and a `BoardPoller` completion transaction. The adapter records completeness only after every page succeeds. Reconciliation writes observations, content-independent last-seen state, closure changes, empty streak, and `last_completed_at` together after ingest acceptance; failure leaves due time intact. A complete nonempty snapshot closes omitted Jobs after one check. A complete empty snapshot closes open Jobs only after two consecutive successful empty checks. When a tracked careers page supersedes a Board, continue checks until two complete empty snapshots, then mark it retired. Snapshot version/fencing prevents a late older check from overwriting a newer result. Manual checks reconcile availability but do not advance scheduled cadence. HTML Jobs close only on their own page's explicit closed signal or confirmed 404/410.

### Acceptance criteria

- [ ] A failed, partial, or persistence-failed check neither closes Jobs nor advances completion freshness.
- [ ] One complete nonempty omission closes a Job; one complete empty result does not; two complete empty results close remaining Jobs and can retire a superseded Board.
- [ ] A later complete appearance reopens a Job without treating it as newly discovered.
- [ ] Replaying a snapshot or completing an older snapshot after a newer one leaves the newer state intact.
- [ ] A partial ingest failure blocks Board completion, and an HTML transient error never closes its Job.
- [ ] Board status and last successful check are visible in Company detail; isolated BoardPoller tests cover each lifecycle path.

---

## Phase 4: Stable Job identity and unchanged-content reuse

**User stories:** As a User, I see one Job when several URLs point to the same trusted ATS posting. As an operator, repeat Board checks do not repeat full-content processing or AI calls for unchanged Jobs.

### What to build

Add provider posting IDs, URL aliases, content fingerprints, Company/Board FKs, and `first_discovered_at`. Resolve a trusted `(Board, posting ID)` before URL; keep unknown IDs provisional by normalized URL. Record every complete Board observation, but send only new or meaningfully changed content to ingest. Ingest returns per-item outcomes and treats identical replay as unchanged; changes to title, description, location, salary, or work arrangement advance the fingerprint and make affected Suitability stale. Preserve latest detail and close/reopen state separately. Batch ATS export through `POST /ingest/batch`, with bounded batch size and per-item outcomes.

### Acceptance criteria

- [ ] Two aliases with the same trusted Board/posting ID resolve to one Job; different providers' IDs do not merge automatically.
- [ ] A second unchanged Board check updates observation/freshness but makes zero additional AI calls or alert attempts.
- [ ] A meaningful content change updates the Job once and identifies affected User assessments; a URL-only change adds an alias without new Job identity.
- [ ] Replaying the same batch is idempotent, including after an ambiguous HTTP response; invalid rows are returned as explicit rejections.
- [ ] The Jobs and Company views show the canonical Job and current availability without duplicate rows.

---

## Phase 5: Recoverable detail and scrape-request work

**User stories:** As a User, a transient fetch or process crash does not silently lose a Candidate or an explicit rerun. As an operator, I can inspect and replay terminal failures.

### What to build

Replace `ZPOPMIN`/`LPOP` consumption with atomic publish and due-time claim/lease/ack for detail Jobs and discovery scrape requests. Preserve card payload through retries, use bounded exponential backoff, reclaim expired leases, and retain dead letters with a replay operation. Ack only after `POST /ingest` durably accepts the item. Deduplicate by normalized URL or request ID while allowing a new explicit rerun. Verify Valkey AOF/backup settings and restart behavior. Expose queue counts, oldest age, and replay to an operator command or authenticated admin surface.

### Acceptance criteria

- [ ] A crash after claim, after HTTP send, or before ack leads to eventual retry and one canonical Job/effect set.
- [ ] Future-due retries are not claimed early, and payload/card data survives every retry and dead-letter transition.
- [ ] Repeated enqueue does not multiply pending work; an explicit later rerun remains possible.
- [ ] A dead letter can be inspected and replayed with its original payload; real-Valkey integration tests cover atomicity and lease expiry.
- [ ] A Valkey restart under the chosen persistence policy preserves acknowledged and pending state as documented.

---

## Phase 6: Durable per-User scoring and first-discovery alerts

**User stories:** As a User, I receive a new-Job alert only after Suitability reaches my threshold, once at first discovery. When I track a Company later, I see its cached open Jobs and get assessments without false new alerts. I can request a rescore after changing my rubric.

### What to build

Move score and notification work out of synchronous `POST /ingest` into the transactional `effect_outbox`. Score by Job fingerprint, User, Search Config version, and model; retry failures independently. Tracking a Company enqueues assessments for its known open Jobs. A Search Config edit marks old assessments potentially stale; only an explicit rescore queues existing Jobs in bounded batches. After a successful qualifying score, create a unique `notification_events` row for first-discovery eligibility and submit email to the configured recipient. Show stale/pending/failed assessment state and rescore action in the UI. Define recipient behavior for the current configured email explicitly; do not imply multi-User email routing that is not implemented.

### Acceptance criteria

- [ ] `POST /ingest` returns after durable Job/outbox commit without waiting for Anthropic or Resend.
- [ ] An unchanged replay creates no new scoring or notification work; changed content scores affected Users once per input version.
- [ ] New tracking shows cached open Jobs immediately and queues their assessments without new-Job alerts.
- [ ] A changed rubric makes prior scores visibly stale; explicit rescore is bounded and deduplicated.
- [ ] A qualifying first-discovered Job sends at most one recorded new-Job event per User; edits, reopening, and Candidate promotion send none.
- [ ] Failed AI or email delivery is retryable and observable without rolling back Job persistence.

---

## Phase 7: Explicit Web Unlocker fetch controls

**User stories:** As an operator, protected sources use the paid path predictably, budget exhaustion pauses only those sources, and direct ATS checks continue. As a User, unsafe or malformed URLs cannot cause the worker to fetch private network resources.

### What to build

Inject a direct or Web Unlocker `Fetcher` into each source adapter. Validate required `BRIGHTDATA_PROXY_URL` at worker startup, establish TLS verification with the provider CA or verified HTTPS, and remove direct fallback for protected sources. Classify zone exhaustion separately from transient rate limiting; pause protected sources, preserve their due work, probe once daily, and resume on success. Add URL/IP/redirect admission, response-size/time limits, per-host concurrency and per-User scrape-now admission. Update the source guide and deployment configuration to match ADR 0009.

### Acceptance criteria

- [ ] Missing or malformed Web Unlocker configuration prevents worker startup; invalid credentials fail protected checks visibly without direct retry.
- [ ] Zone exhaustion pauses protected sources, one daily probe resumes them after recovery, and direct ATS checks continue during the pause.
- [ ] Temporary rate limits use bounded retry without a global zone pause.
- [ ] Private-address, redirect, oversized-response, and unsafe-scheme cases are rejected for both access paths.
- [ ] Fetcher tests prove source parsing is the same under direct and Web Unlocker transports.

---

## Phase 8: Bounded throughput and paginated reads

**User stories:** As a User, Jobs and Company pages stay responsive as the catalog grows. As an operator, collection throughput rises under a controlled provider budget rather than unbounded parallel requests.

### What to build

Run a bounded detail pool with per-host/provider limits and remove the fixed 10–15 second global item sleep. Batch ATS ingest with explicit size and payload limits. Add cursor pagination, availability filter, and stable ordering to `GET /jobs`, and adapt Jobs, Company detail, and Insights reads so they do not fetch every description to render lists. Keep full detail behind `GET /jobs/{id}`. Index the corresponding queries and record latency, queue age, and paid-request efficiency.

### Acceptance criteria

- [ ] Concurrent details respect global and per-host caps and drain a representative backlog faster than the single delayed consumer without exceeding configured provider limits.
- [ ] A Board with many unchanged Jobs sends bounded batches and does not create one HTTP request per returned Job.
- [ ] Job list and Company detail use cursor pages with stable next-page behavior under inserts; full description loads only for detail views.
- [ ] Query plans and load measurements are recorded for the target catalog size, with no full-table response from `/jobs`.

---

## Phase 9: Measured rollout and operational proof

**User stories:** As an operator, I can tell whether collection is late, paid fetching is paused, Jobs are stuck, or AI cost is rising; I can deploy safely at the agreed scale and restore service after failure.

### What to build

Add a small operations view or command for Board lag, queue/dead-letter state, protected-source pause, failed effects, and replay. Instrument source completeness/failure, Candidate conversion, unchanged-hit ratio, AI calls/cost, notifications, and Web Unlocker requests. Run migration/backfill checks and a representative load test for one or a few Users tracking up to hundreds of Companies. Keep one active scheduler for this rollout; test atomic Board claims and overlapping ticks before allowing scheduler replicas. Document recovery procedures for PostgreSQL, Valkey, expired leases, exhausted Web Unlocker zone, and interrupted migrations; correct README/system overview claims that the rollout changes.

### Acceptance criteria

- [ ] A repeat unchanged Board run produces zero AI calls and new alerts; a failed/partial run produces zero false closures.
- [ ] Two concurrent scheduler attempts produce one accepted Board claim and one completion; the single-scheduler deployment remains the default.
- [ ] Load-test results state Board count, Job count, Users, queue latency, poll lateness, API p95, AI calls, paid requests, and hardware/configuration; capacity claims are based on those results.
- [ ] Operator metrics or commands expose oldest pending work, expired leases, dead letters, Board failures, zone pause, and failed outbox effects with replay paths.
- [ ] Backfill counts, ambiguous identity cases, restore exercise, and rollback procedure are recorded before retiring legacy columns or enabling additional replicas.
