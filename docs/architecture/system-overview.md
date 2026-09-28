# System overview

[Editable Excalidraw diagram](system-overview.excalidraw) · [PNG preview](system-overview.png) · [SVG export](system-overview.svg)

Reviewed against the working tree based on `8946bd5` on 2026-09-23, including the existing local orchestrator change. The diagram contains eleven components and fourteen relationships. It shows current runtime paths and the agreed intent for detail reuse and selected-company polling. The boundary denotes the logical product, including its browser SPA; it does not imply a single host or network. Solid arrows show synchronous calls or user actions, with responses implicit. Dashed amber arrows show work queued for later processing; the underlying Valkey commands themselves are synchronous.

The user browses jobs and scores, tracks applications and companies, and manages CV references through the SolidJS SPA and Go API. The worker discovers companies and scrapes configured sources on a schedule or in response to queued scrape requests. Sources returning full jobs export directly to the API; HTML sources enqueue URLs for detail fetching first. Both paths converge on authenticated `POST /ingest`. The API saves jobs, scores them for matching users with stored credentials, and optionally submits new-job emails, all before the ingest call returns.

## Collection model and intended cost boundary

Currently, the shared company catalog records one detected ATS provider and board token. Detection alone does not enable recurring job collection. A user selects/tracks a company by creating an enabled source target; scheduled checks fetch only due enabled boards. The builder deduplicates the same `(source, board token)` across users within a run. Careers discovery can still enrich unselected companies without polling their job inventories. The agreed future model makes tracking a Company-level choice covering all current and later Verified Boards, including when none is known yet (ADR 0007).

The intended invariant is **fetch expensive job details once per meaningful content version, then reuse them across users**. Periodic cheap board inventories remain necessary to discover new postings and reconcile availability. A job can close, reopen, or change salary/location/description; “scrape once” is a cost policy, not permanent immutability. Prefer provider IDs plus board identity, with canonical URL aliases, and retain last-seen/availability independently of stored detail content. Keep the latest details, change timestamps, and a content fingerprint rather than every full historical version. Use provider update timestamps or content hashes where available; refresh details only on new identity, a meaningful change, or an explicit stale-data check. Greenhouse, for example, exposes posting IDs and `updated_at` in its [public Job Board API](https://docs.greenhouse.io/job-board.html).

Trusted ATS identity can merge different URLs for the same posting. Automatically matching a vacancy across different ATS providers after a Company migration is outside the near-term scope; distinct provider IDs remain distinct Jobs. The application window is short, so the extra matching workflow is not justified now.

| Concern | Current behavior | Intended refinement |
| --- | --- | --- |
| Company → ATS mapping | Harvest/crawl and ingest populate the shared catalog; tracking enables a target. | Periodic YC/Getro discovery surfaces Companies in the Companies view without inspecting untracked careers sites. Tracking triggers an initial careers-site inspection; Users can also add verified Boards manually. Recheck tracked Companies' careers sites about monthly and sooner after repeated Board failures to detect migrations. |
| Discovery cadence | YC/Getro company harvesters and configured job-board searches recur. | Keep periodic Company-catalog harvesting, but run each job-board discovery search once when requested or explicitly rerun; retain its Job Candidates for later interest. |
| Selected ATS polling | Due target intervals, within-run board deduplication, direct API fetches. | Periodically check only Verified Boards of Tracked Companies at the shortest requested Check Frequency. Manual checks leave the scheduled cadence intact. One active scheduler suffices at the agreed near-term scale; lease Boards before adding scheduler replicas. |
| HTML details | Known exact URLs are skipped before queueing. | Retain cheap rejected Job Candidates for 60 days, refreshing the window on explicit rediscovery. Re-evaluate them in bounded background batches when a discovery Target or cheap criteria change; only candidates passing the current gate proceed to detail fetching. Use stable identity and URL aliases; atomic enqueue plus lease/ack retries after worker crashes, with idempotent ingest. Do not automatically refetch stored HTML details. |
| ATS content | Full returned jobs are exported/upserted on every poll. | Separate inventory observations from new/changed content; unchanged jobs avoid repeated scoring, and new-Job alerts occur only at first discovery after a User's suitability threshold is met, never on edits, reopening, or later Candidate promotion. |
| Job lifecycle | Scheduled successful nonempty snapshots close missing jobs; upserts reopen jobs. | Require one complete nonempty check for a missing ATS Job or two successful complete empty checks for its Board; close an HTML Job only on its page's explicit closed signal or confirmed 404/410. Transient errors do not close Jobs; prevent old snapshots overwriting newer state. |
| Personalisation | Synchronous scoring for matching tracking users during ingest. | Durable asynchronous scoring keyed by job version, user, and scoring-input version; a User tracking a Company sees its known open Jobs immediately and receives queued scores without new-Job alerts. |

## Bright Data fetch boundary

The new external-provider box makes paid fetching visible. Cooperative ATS APIs and company discovery stay on the direct path. Selected protected HTML sources use the existing Bright Data Web Unlocker proxy transport (`BRIGHTDATA_PROXY_URL`); Bright Data manages unblocking and proxy infrastructure. The intended provider is Web Unlocker, using the existing native proxy access path. It handles proxy selection and unblocking for those requests. See [Web Unlocker access modes](https://docs.brightdata.com/products/web-unlocker/introduction).

**Web Unlocker is the only intended Bright Data integration.** No managed Web Scraper API integration is planned. ADR 0003 records the provider decision; the diagram's paid path is the implemented Web Unlocker proxy transport, not a new data-source or parser contract.

Choose a direct or Web Unlocker fetcher per source. The agreed worker startup contract requires Web Unlocker credentials even when current sources are direct-only; missing configuration prevents startup. Wrong or expired credentials fail the affected source check, with no direct fallback. Bright Data's zone spending limit is the only spending guard. A zone-limit error pauses Web Unlocker sources; one daily probe resumes them when access returns, while direct ATS and cooperative sources continue. Temporary rate limits get bounded retries rather than a zone-wide pause. The current transport does not yet implement this: it falls back to direct when configuration is missing or invalid and disables proxy TLS verification. Enforce per-host/provider concurrency at the fetch boundary. See ADR 0003.


## Supporting repository files

| Evidence | What it establishes |
| --- | --- |
| [README](../../README.md), [domain context](../../CONTEXT.md), [frontend README](../../frontend/README.md) | Product purpose and terminology; discrepancies are recorded below. |
| [Compose](../../docker-compose.yml), [Dockerfile](../../Dockerfile) | Separate API and worker processes; PostgreSQL 17 and Valkey 8 with mounted data volumes; no frontend deployment service. |
| [Frontend entry](../../frontend/src/main.tsx), [API client](../../frontend/src/api/client.ts), [API configuration](../../frontend/src/api/config.ts), [Vite configuration](../../frontend/vite.config.ts) | SolidJS browser SPA; direct fetch calls to the configured API with cookies; no configured Vite API proxy. |
| [API entry](../../cmd/api/main.go), [router](../../internal/api/router.go) | HTTP listener, PostgreSQL and Valkey clients, session-protected user routes, service-token ingest, Google/AI/email wiring. |
| [Worker entry](../../cmd/worker/main.go), [orchestrator](../../internal/worker/scraper/orchestrate.go), [worker loops](../../internal/worker/worker.go) | Scheduling, scrape-now consumption, HTML detail fetching, company harvesting/crawling, direct DB reads and writes, session cleanup. |
| [Queue](../../internal/queue/queue.go), [source-target handler](../../internal/api/handlers/source_targets.go), [company handler](../../internal/api/handlers/companies.go) | Pending URL sorted set, scrape-request list, retries/dead letters, API enqueue operations. |
| [Source builder](../../internal/worker/sources/builder/build.go), [source dispatch](../../internal/worker/sources/source.go), [proxy transport](../../internal/worker/proxy/proxy.go) | Target-driven source selection, detail-fetch routing, optional BrightData transport. |
| [API exporter](../../internal/worker/scraper/api_exporter.go), [ingest](../../internal/api/ingest/ingest.go), [DB connection](../../internal/data/conn.go) | Worker-to-API HTTP/JSON with bearer token; synchronous ingest sequencing; PostgreSQL via pgx. |
| [Claude client](../../internal/score/claude.go), [ingest scorer](../../internal/score/ingest.go), [reasoning handler](../../internal/api/handlers/job_reasoning.go) | Per-user suitability scoring, persisted scores, on-demand reasoning using Anthropic. |
| [Google client](../../internal/api/google/client.go), [CV service](../../internal/cvtemplates/service.go) | OAuth token exchange/refresh, live Docs tabs, Drive metadata, HTTPS PDF export. |
| [Notification service](../../internal/api/notify/service.go), [Resend client](../../internal/api/notify/resend.go) | Optional ingest email submission to a configured recipient; digest method exists but has no runtime caller. |

## Material documentation disagreements

- **Worker database access:** the README says the worker never touches the DB directly. Its entry point creates a DB pool, reads targets and search criteria, maintains company/job state, and deletes expired sessions. The diagram includes this connection.
- **Notification behavior:** the README describes a scheduled digest and suitability-gated notifications. Neither entry point schedules `SendDigest`; ingest passes score `0` to notifications, and router wiring leaves `NotifyThreshold` at its zero default. Only optional new-job email submission is shown.
- **Scoring configuration and timing:** the README describes `SCORING_USER_ID`/`ANTHROPIC_API_KEY` setup and a numeric relevance gate. Current wiring uses stored per-user credentials and matching source targets for suitability, with a multi-user reject filter during scraping. Although ingest comments say “fire-and-forget” and “never block,” the calls are synchronous; errors after saving are handled internally rather than returned.
- **Sources:** WIS is described as always active, but `BuildSources` only schedules it when configured targets exist. The worker's unconditional bare WIS instance is for detail fetching. CONTEXT describes aggregators as discovery-only and WIS as the only filter source; current LinkedIn/Indeed adapters implement detail fetching, and the builder also supports LinkedIn, RemoteOK, and Remotive filters. The diagram uses the verified full-job versus detail-fetch capability split.
- **Frontend transport:** the frontend README claims Vite proxies API traffic. The Vite configuration contains no proxy, and the API client uses `VITE_API_URL` or `http://localhost:8080` directly.

## Scope and unresolved assumptions

- Production SPA hosting, public DNS, TLS termination, and any reverse proxy are unspecified. The Go API serves routes, not the SPA assets. No unverified gateway or hosting service is drawn.
- This is configured capability, not an audit of running infrastructure. Deployment-specific integration availability and the actual email recipient are not asserted. The recipient is a configured address, not necessarily the signed-in user.
- Bright Data is an explicit external dependency. Its provider-to-site arrow summarizes fetching on our behalf; it does not promise a particular IP class for every request. Web Unlocker proxy use is currently optional. Company discovery, ATS APIs, and HTML sources retain one external-system group.
- Browser OAuth redirects and individual source adapters are omitted to preserve the overview level of detail; Google token exchange and CV access are grouped into one API-to-Google connection.
- PostgreSQL and Valkey have Compose volumes. The diagram does not assert a queue durability or delivery guarantee: no explicit Valkey AOF policy is configured, and scrape-now failures are best effort.
- CV content remains in Google Docs; PostgreSQL stores references, tab visibility, and encrypted OAuth tokens. Browser demo mode and browser-local display preferences are outside the production flow shown.

## Validation

The scene's JSON, unique IDs, paired shape/text bindings, arrow endpoints, and component/relationship counts were checked. The scene was exported with Excalidraw's renderer and visually inspected as a PNG. The editable scene, SVG, and PNG represent the same diagram. Only architecture documentation and diagram artifacts were changed; the pre-existing application-code edit was preserved.
