# Approach

_Last updated: 2026-06-14 · Canonical alignment doc — what we are building and why._

## Background Research

### 1. Use-case frame

You have a working two-phase scraper — discovery (list pages → URLs) feeding a Valkey queue,
and extraction (one worker pulls a URL, fetches the detail page, parses, upserts). Today it
runs a single hardcoded HTML source (Work In Startups, `q=product+engineer&w=uk`) with a
static User-Agent, no proxies, and no relevance filtering. The goal is to evolve this into an
*intelligent, selective* system: pull from cheap structured sources by default, spend
expensive proxy-backed HTML scraping only where there is no alternative, and decide *what* is
worth fetching before paying to fetch it. Crucially, in your domain a job is scraped **once** —
there is no price-watching re-scrape loop — so "incremental" here means *never re-doing work*,
not *detecting change*.

### 2. The mental model (key concepts)

1. **Pipeline, not a bot.** The strongest framing across the sources (Olostep): a job scraper
   is a multi-stage data pipeline — *discovery → extraction → normalization → dedup → freshness
   → delivery* — not one script that "scrapes a site." You already have discovery/extraction
   split cleanly; the work ahead is making the middle stages explicit. *Why it matters: it tells
   you where each new capability (filtering, API sources, validation) belongs.*

2. **Source tiering.** Sources are not equal in cost or reliability. Tier 1 = company ATS
   endpoints (structured JSON, no auth, no proxies — the source of truth). Tier 2 = aggregators
   (LinkedIn/Indeed) used for *URL discovery only*. Tier 3 = general job boards as fallback.
   *Why it matters: the entire "only scrape expensively when strictly necessary" goal is just
   "prefer lower tiers," made operational.*

3. **API-first extraction.** Greenhouse, Lever, Ashby, Workable, Recruitee and Personio all
   expose **public, unauthenticated JSON (or XML) job-board APIs**. For these you don't scrape
   at all — and `?content=true` style params return the *full description in the list call*,
   collapsing your two-phase fetch into one cheap request with zero blocking risk. *Why it
   matters: a large share of real jobs sit behind one of these ATSs; routing to them eliminates
   proxies and HTML parsing entirely for that slice.*

4. **Pre-extraction filtering (the gate).** The cheapest request is the one you never make.
   Multi-stage filtering means scoring a candidate on the *cheap* signals you already have
   (URL, listing-card title/company/location) and only promoting survivors to the expensive
   detail fetch. *Why it matters: this is the lever that makes the system "selective"; it lives
   exactly where your `NewURLs` dedup already sits.*

5. **Structured-data-first parsing.** Prefer machine-meant data over human-meant markup:
   JSON-LD (`<script type="application/ld+json">`), Open Graph, or a site's own internal JSON/XHR
   endpoints. These break far less often than CSS selectors and need little cleaning. *Why it
   matters: it's both more robust and cheaper to parse than the `goquery` selector approach in
   `wis.go`.*

6. **Proxy tiering.** Proxies are a spectrum, not on/off. Datacenter IPs are cheap but flagged
   by default on Indeed/LinkedIn; residential IPs are expensive but necessary for hostile
   aggregators. The discipline: route ATS/API calls direct or via datacenter proxies, and
   *reserve residential bandwidth strictly for Tier-2 aggregators.* *Why it matters: "only when
   strictly necessary" applies to proxy *grade* as much as proxy *use*.*

7. **Scrape-once / delta discovery.** Your dedup key (`jobs.url UNIQUE` + `NewURLs`) already
   guarantees a URL is fetched once. The incremental-scraping literature adds one more saving on
   the *discovery* side: stop paginating as soon as a page yields no new URLs (you already do
   this in `orchestrate.go`), and use cheap change-checks (sitemap / `Last-Modified` / `ETag`)
   to decide whether to even start a discovery pass. *Why it matters: in a scrape-once world
   these are your only re-visit costs, so they're where discovery savings live.*

8. **Silent failure & data accuracy.** The dangerous scraper failure is the one that *succeeds*
   — returns a 200, parses "something," and stores wrong/empty data. Layout changes that move
   data into the wrong container, silent fallbacks (defaulting a missing field), and partial
   extraction all corrupt data without erroring. *Why it matters: your `ParseJobDetail` already
   has a silent fallback (defaults `UpdatedAt` to `time.Now()` when the date is missing) — the
   exact anti-pattern PromptCloud warns about.*

### 3. Consensus view

The sources converge hard on a few points:

- **Prefer official/ATS APIs over scraping, always.** This is unanimous where it comes up.
  Public ATS APIs (Greenhouse et al.) give you clean, structured, complete data with no auth,
  no proxies, and no parser maintenance. Teams reportedly burn 30–40% of data-engineering time
  just *maintaining* HTML scrapers — cost you avoid entirely by using an API.

- **Route by source type before doing work.** Detect the ATS/provider from the URL and dispatch
  to the cheapest viable method (direct JSON API → datacenter proxy → residential proxy +
  browser → AI extraction fallback). Don't run one uniform expensive path for everything.

- **Filter and dedup early, before expensive stages.** Every production write-up gates work:
  drop already-seen items (URL/UID sets), batch AI/expensive steps, and use `INSERT … ON
  CONFLICT / IGNORE` to avoid redundant downstream processing. You already do the URL-dedup
  half of this.

- **Reserve residential proxies for the hostile aggregators only.** Datacenter IPs are fine for
  APIs and friendly sites; LinkedIn/Indeed need residential IPs and behavioral care. Nobody
  recommends proxying everything.

- **Stop early / go incremental.** Halt pagination when you hit known items; use conditional
  requests and last-modified checks so you don't re-fetch unchanged surfaces.

- **Validate output and monitor for silent failure.** Schema/Pydantic-style validation at the
  extraction boundary, plus run-over-run monitoring of success rates, item counts and status
  codes (ScrapeOps' whole pitch), because "the job ran" ≠ "the data is right."

### 4. Tensions and trade-offs

- **Scrape the aggregators at all? (the biggest decision.)** ApplyArc's blunt finding — 3 of 5
  LinkedIn scrapers got flagged within ~200 pulls; "selective saving beats aggressive scraping"
  — sits against ScrapFly/Voyager/JobSpy, which show LinkedIn *is* scrapable via hidden JSON and
  the `seeMoreJobPostings`/Voyager endpoints. The reconciliation that fits your project: LinkedIn
  job *listings* have a lower detection bar than profiles, but still need residential IPs and
  rate discipline. For a personal/low-volume job tool, treat aggregators as Tier-2 *discovery*
  (find the URL, then fetch the real posting from the company ATS) rather than a primary
  extraction target — you get the coverage without owning the blocking arms race.

- **AI categorization vs cheap heuristics for the filter stage.** Hesam's aggregator runs
  GPT-4o-mini over job *titles* in 150-item batches as a daily cron — cheap because it's
  batched, title-only, and post-dedup. That's a viable relevance filter, but for a single fixed
  intent ("product engineer, UK") keyword/heuristic scoring on the listing card is near-free and
  may be enough. The trade: LLM categorization scales to fuzzy/multi-intent matching; heuristics
  are free and deterministic. Start heuristic, add LLM only if your matching genuinely needs it.

- **Self-healing extraction: deterministic vs LLM.** Anansi heals broken selectors *without*
  LLMs — structured-data-first, then regex/fuzzy-class/structural/XPath fallbacks with
  confidence scoring — explicitly to avoid LLM latency/cost. HasData reaches for an AI-extraction
  fallback on unknown domains. For a small, known set of sources, anansi's "JSON-LD first, CSS as
  scored fallback" is the higher-leverage pattern; LLM extraction is a fallback for the long tail
  you don't want to hand-write parsers for.

- **One rich API call vs two-phase fetch.** Your architecture is two-phase (list → queue →
  detail) because HTML list pages don't carry full data. But ATS APIs *do* (`content=true`).
  For API sources the two-phase split is unnecessary overhead — you can fetch list+content in
  one call and skip the queue. Decide per-source whether phase-2 is even needed, rather than
  forcing every source through the same pipeline.

### 5. Practical patterns for your use case

Ordered by leverage for *your* code:

1. **Add ATS API sources behind your existing `Source` interface.** Your `source.go` `Source`
   interface + `Dispatch`/`CanHandle` prefix routing is already the right seam. A Greenhouse
   source is roughly: `GET https://boards-api.greenhouse.io/v1/boards/{token}/jobs?content=true`
   → iterate the JSON array → emit fully-populated `dto.Job`s directly. No auth, no proxy, no
   `goquery`. Lever (`api.lever.co/v0/postings/{site}?mode=json`), Ashby
   (`api.ashbyhq.com/posting-api/job-board/{name}?includeCompensation=true`), Workable, Recruitee
   and Personio (XML) follow the same shape. These become your Tier-1 default.

2. **Make extraction-need a per-source property.** For API sources, `GetDetails` is a no-op —
   the list call already returned everything. Let a source declare whether it needs phase-2
   detail fetching, so API sources bypass the queue/worker detail path entirely and only HTML
   sources pay for it.

3. **Insert the filter gate at `orchestrate.go` (post-`NewURLs`, pre-`Enqueue`).** This is the
   single highest-leverage change for selectivity. It runs *before* anything expensive. To make
   it useful you must first **enrich the listing parse to keep card metadata** — right now
   `wis.go`'s `ParseURLs` throws away everything but the href, even though the cards contain
   title/company/location. Return a partial `dto.Job` per card, score it (keyword/heuristic now,
   LLM later), and only enqueue survivors. Multi-stage = (a) URL filter you have → (b) new
   listing-metadata filter here → (c) optional full-text filter after detail fetch in the worker
   handler.

4. **Tier your proxy use, and make it source-config.** Add an optional proxy/transport to
   `PaginatedBase.Get`, defaulted off. APIs and friendly HTML: direct or datacenter. Only a
   source explicitly marked as a hostile aggregator routes through residential IPs. This keeps
   "expensive proxying only when strictly necessary" enforced by configuration, not by habit.

5. **Structured-data-first parsing for HTML sources.** Before reaching for CSS selectors, check
   for JSON-LD `JobPosting` blocks (most modern job pages embed one for Google indexing) and
   internal JSON. This is more robust than `goquery` selectors and is the anansi/ScrapFly
   consensus. Keep CSS as a scored fallback.

6. **Discovery via search/sitemaps to reach Tier-1 sources.** HasData/Olostep: use Google
   `site:boards.greenhouse.io "<role>"` or ATS sitemaps as a *discovery* layer that yields clean
   company-ATS URLs, sidestepping aggregator anti-bot entirely. This is how you get LinkedIn-grade
   coverage while still extracting from cheap Tier-1 endpoints.

7. **Fix requeue-on-failure (correctness, not selectivity).** `worker.go` swallows handler
   errors and the URL is already `ZPOPMIN`'d, so a transient detail-fetch failure permanently
   drops the job (it's recoverable only because no DB row was written, so a later discovery pass
   re-surfaces it — fragile). Add a bounded retry / dead-letter so transient blocks don't silently
   lose jobs.

8. **Validate at the extraction boundary + monitor runs.** Reject/flag a parsed `dto.Job` whose
   required fields are empty rather than storing a husk, and *log* every fallback (the
   `time.Now()` date default especially) so silent fallbacks become visible. Track per-source
   success rate, item counts and status codes across runs (ScrapeOps-style) — for a scrape-once
   system, a silent drop in a source's yield is your main failure mode and won't error.

### 6. What to watch out for

- **Silent fallbacks corrupt data quietly.** Your `ParseJobDetail` defaults `UpdatedAt` to
  `time.Now()` when no date is found — PromptCloud's exact anti-pattern. Untracked, you can't
  tell real timestamps from substituted ones. Tag or log every defaulted field.
- **"200 OK" ≠ "good data."** Anti-bot pages, soft 404s, and layout changes return 200 with
  wrong/empty content. Validate semantics, not just HTTP status.
- **Datacenter IPs are pre-flagged** on Indeed/LinkedIn (AWS/GCP/Azure ranges). Don't assume a
  cheap proxy will work on aggregators — it's the one place you actually need residential.
- **`MockQueue.Enqueue` doesn't replicate the real `ZADD NX` dedup**, so tests can pass while
  masking duplicate-enqueue behavior. Keep that divergence in mind when testing the filter gate.
- **No `description`/`salary` columns exist** in the `jobs` table. Any full-text or salary-based
  filter — or populating the `Remuneration` field the email template already references but
  never fills — needs a schema migration first.
- **ATS APIs have no search/filter server-side.** Greenhouse/Ashby return *all* of a board's
  jobs; you filter client-side. Cheap, but you fetch the whole board to find the few you want.
- **Aggregator scraping is an arms race with account/legal risk.** ApplyArc's data says
  aggressive LinkedIn scraping gets flagged fast. Favour discovery-only use of aggregators.

### 7. Gaps in the reading list

- **No Go-specific guidance.** Every production example is Python (`curl_cffi`, Playwright,
  FlashText). The patterns port, but you'll be choosing Go equivalents (JSON-LD parsing,
  TLS-fingerprint handling, a proxy transport) yourself.
- **No cost modelling.** Sources say "residential is expensive" but give no numbers to decide the
  proxy-tier threshold for a low-volume personal tool.
- **How to *discover which companies/board tokens* to hit isn't covered.** The ATS APIs are
  per-board (`{board_token}`); none of the sources explains assembling and maintaining that list
  of companies — the practical bootstrapping problem for a Tier-1-first system.
- **MindStudio "6 Token-Saving Techniques" (404, unreachable).** Intended to cover HTML→token
  reduction before LLM calls. Partly compensated by Hesam (title-only batching) and anansi
  (structured-data-first), but if you go the LLM-filtering route the specific cleaning/chunking
  techniques are a gap to fill elsewhere.
- **Legal/ToS and ghost-job detection** are mentioned only in passing (Olostep's "freshness
  scoring" for expired/ghost listings) without concrete method.

### 8. Recommended next steps

- **Build one Greenhouse API source end-to-end** behind the existing `Source` interface as the
  Tier-1 proof: `boards-api...?content=true` → fully-populated `dto.Job`, no queue/detail phase,
  no proxy. This validates the "API source that skips phase-2" shape before generalising to
  Lever/Ashby/etc.
- **Decide the filter contract before coding it:** what signals (URL only, or enriched listing
  metadata) and what method (heuristic keywords vs batched LLM). Recommended: enrich `ParseURLs`
  to keep card title/company/location, start with heuristic scoring at the `orchestrate.go` gate,
  leave an LLM seam for later.
- **Add a per-source `needsProxy` / transport config** to `PaginatedBase`, defaulted off, so
  proxy cost is opt-in per source rather than global.
- **Add a `jobs` schema migration** for `description` (and optionally `salary`) if you want
  full-text/salary filtering or to populate the email `Remuneration` field — decide this early
  since it gates the second filter stage.
- **Fix the requeue/dead-letter gap in `worker.go`** and add fallback-logging in `ParseJobDetail`
  before scaling sources — both are cheap correctness wins that get more valuable as source count
  grows.

## Problem

The current scraper is a single hardcoded HTML source (Work In Startups, locked to a startup-job query) that fetches the full detail page for every discovered URL with no relevance check and no knowledge of ATS APIs. As coverage grows, this becomes expensive: proxy costs scale with every scrape, irrelevant jobs pollute the digest, and HTML parsers break silently when sites change. The system needs to become both *general* (searching whatever the user configures, not only startups) and *selective* — a cheap heuristic relevance gate that filters on listing-card metadata before paying for any detail fetch, a cheap LLM suitability score computed on ingest once the expensive scrape is already done, and structured ATS API sources that eliminate proxy cost entirely for the jobs they cover.

## Goals

- Pull relevant jobs across whatever role, location, and criteria the user configures — startups are the origin use case, not a constraint
- Decide what to scrape before scraping it — cheap heuristic relevance gate on listing-card signals first, expensive detail fetch only for survivors
- Eliminate proxy cost for any job that can be fetched from a structured ATS API
- Score every ingested job 0–100 for suitability against a rubric derived from the user's criteria, so jobs can be ranked, filtered, and gated for notification
- Store description and salary so filtering, scoring, and email digests have the full picture
- Keep scrape-once semantics: a job URL is fetched exactly once, never revisited

## Non-goals

- Multi-user support — single recipient, single intent for now
- Re-scraping or change detection — job postings don't need staleness tracking

## Features

Each feature is a user-facing capability. No priority tiers — use `approach-to-roadmap` for Now/Next/Later.

### ATS URL detection and routing

- **Problem:** Discovered job URLs may point to any of several ATS platforms; without detection the system treats all URLs as HTML targets and misses free API paths.
- **Value:** Unlocks API-first extraction — a URL recognised as `boards.greenhouse.io` gets structured JSON treatment; only unrecognised or hostile sources fall back to HTML scraping.

### ATS API sources (Greenhouse, Lever, Ashby, Workable, Recruitee, Personio)

- **Problem:** A large share of jobs (across any sector) are hosted on one of five or six ATS platforms that expose public, unauthenticated JSON APIs, but the system currently HTML-scrapes everything.
- **Value:** Full description and salary in a single cheap request, no proxy, no HTML parser fragility. Collapses the two-phase discovery+detail flow into one call for these sources.
- **Dependencies:** ATS URL detection and routing

### Listing-metadata enrichment

- **Problem:** `ParseURLs` discards all card metadata (title, company, location) and returns only the URL, so there is nothing to filter on before the expensive detail fetch.
- **Value:** Gives the relevance filter gate real signals to work with; improves partial-job data quality as a by-product.

### Relevance filter gate

- **Problem:** Every discovered URL goes straight to the detail-fetch queue with no relevance check, wasting scrape budget on irrelevant roles.
- **Value:** Blocks irrelevant jobs from ever reaching the expensive detail-fetch stage, keeping proxy cost and noise low as source count grows. Produces a persisted `relevance_score` used both to gate the detail fetch and as a sortable signal in the UI. Heuristic keyword scoring in v1; scoring logic is swappable without moving the gate.
- **Dependencies:** Listing-metadata enrichment (needs title/company/location to score against)

### Aggregator discovery (Tier 2)

- **Problem:** Existing discovery is limited to what WIS surfaces; jobs listed only on LinkedIn or Indeed are missed entirely.
- **Value:** Broader coverage without owning the aggregator extraction arms race — aggregators are used for URL discovery only, and the actual job content comes from the clean ATS API or a polite direct fetch.
- **Dependencies:** ATS URL detection and routing (so aggregator-sourced URLs are routed correctly)

### Per-source proxy tiering

- **Problem:** There is no proxy infrastructure at all today; adding aggregator sources requires it, but routing every source through residential proxies would be needlessly expensive.
- **Value:** Enforces "proxy only when necessary" in code: API sources run direct, friendly HTML boards run via datacenter proxy, hostile aggregators via residential. Proxy cost is opt-in per source via config.

### Description and salary persistence

- **Problem:** The `jobs` table has no `description`, `salary`, or score columns; full-text filtering is impossible, the `Remuneration` field in email digests is always empty, and suitability scores have nowhere to live.
- **Value:** Enables full-text relevance filtering; populates salary in the digest; provides the `relevance_score` and `suitability_score` columns that the scoring pipeline and UI depend on. All added in a single migration alongside `description` and `salary`.

### LLM suitability scoring (on ingest)

- **Problem:** After an expensive detail scrape the system stores a job but has no signal for how well it actually matches what the user is looking for, so notification and ranking are blind.
- **Value:** A cheap LLM (Claude Haiku) call runs in the worker ingest path immediately after `db.Save`, produces a 0–100 `suitability_score` against a rubric derived from the user's configured criteria, and persists it on the job. Score is stored on every ingested job; the expensive scrape work is already done at this point so there is no cost reason to skip.
- **Dependencies:** Description and salary persistence (rubric uses full job text and salary), Configurable search criteria (criteria are the rubric source)

### Score-ranked job surfacing

- **Problem:** The existing jobs list (`GET /jobs` → `JobsDataTable`) shows all jobs ordered by `scraped_at DESC` with no scoring signal, so users have no way to prioritise or filter by how good a fit a job is.
- **Value:** The jobs list gains relevance and suitability score columns; tanstack sort/filter (already wired client-side) lets the user sort by either score or filter to only high-suitability roles.
- **Dependencies:** Relevance filter gate (relevance score), LLM suitability scoring (suitability score)

### Suitability-gated notifications

- **Problem:** Every ingested job triggers a notification regardless of how relevant it is, creating noise that trains the user to ignore alerts.
- **Value:** `NotifyNewJob` and the daily digest cron check `suitability_score` against a threshold; only jobs above the threshold produce a notification. High signal: the user only hears about genuinely strong matches.
- **Dependencies:** LLM suitability scoring

### Worker dead-letter / bounded retry

- **Problem:** Failed detail fetches are silently swallowed in `worker.go` after the URL has already been popped from the queue, so transient failures lose jobs permanently until accidental rediscovery.
- **Value:** Correctness guarantee: no silent job loss on transient network errors or temporary blocks.

### Configurable search criteria (extensibility target)

- **Problem:** Role, location, and keyword criteria are hardcoded; changing intent requires a code change and redeployment.
- **Value:** Allows search criteria to be adjusted at runtime without touching code; prerequisite for any future multi-user support. Criteria are job-type-agnostic — not startup-specific — and serve as the single source of truth for both the heuristic relevance gate and the LLM suitability rubric.
- **Dependencies:** Relevance filter gate, LLM suitability scoring (criteria drive both)
- **Note:** v1 ships with the user's own static config (not startup-hardcoded); this feature makes that config editable at runtime. Every scoring abstraction must accept criteria as an injected dependency from the start.

## Constraints

- Go backend; existing package layout (`cmd/worker`, `internal/scraper`, `internal/sources`, `internal/queue`, `internal/data`) is the extension surface — no package restructuring
- Valkey sorted-set queue with `ZADD NX` dedup; scrape-once guarantee must be preserved
- Postgres with sqlc; schema changes via numbered migrations in `scripts/migrations/`
- Single recipient (`NOTIFY_EMAIL_TO`) and single configured intent for v1 — no multi-tenancy; criteria are not startup-specific
- Suitability scoring uses Claude Haiku (or equivalent cheap model); must remain low-cost per job — it runs synchronously on every ingest
- Bun for all JS/TS (frontend, emails); pnpm/npm/yarn never

## Principles

- **Domain-agnostic:** the system searches whatever the user configures; startups are the origin use case, not a baked-in assumption — no hardcoded role, sector, or job-type anywhere in the pipeline
- **Two-stage filtering:** cheap heuristic relevance gate on listing-card signals pre-scrape → expensive scrape → cheap LLM suitability score on full text post-scrape. Each stage costs less than the next; the gate's job is to protect the expensive stages
- **Scrape-once:** a job URL is detail-fetched exactly once; the `jobs.url UNIQUE` constraint and `NewURLs` dedup are the enforcement mechanism — don't weaken them
- **API-first:** always prefer a structured ATS API over HTML scraping for the same content
- **Proxy as last resort:** only proxy-back a source that has no API alternative and actively blocks direct requests; tier proxy grade to source hostility
- **Design for extensibility:** v1 ships with static criteria config but every abstraction (filter gate, source router, scoring rubric, proxy transport) must accept that config as an injected dependency, not as a hardcoded value
- **Fail loudly:** no silent fallbacks; log or reject every defaulted or empty field at the extraction boundary

## Open questions

1. Which aggregators for Tier 2 discovery first — LinkedIn and/or Indeed, or a lower-friction alternative (Otta, Wellfound)? Matters because LinkedIn has a lower detection bar for job listings than profiles but still needs residential IPs and careful rate limiting.
2. Salary normalisation at storage time: store raw string from the source, or normalise to annual GBP on ingest? Affects schema design and how comparable cross-source salary data is.
3. ATS URL detection architecture: implement as a lookup inside `Dispatch` (self-registration by each source via `CanHandle`), or a dedicated `Detect(url) ATSType` step that runs before dispatch? The latter is cleaner for the aggregator path where the source isn't known at discovery time.
4. Suitability notification threshold: what default score (e.g. 70/100) and should the user be able to adjust it without a code change? Matters for how soon the feature delivers value out of the box.
5. LLM suitability model: Claude Haiku assumed for cost — confirm this and set a per-job token/cost ceiling so the scoring step is auditable.

## Out of scope

- Re-scraping or staleness detection — job postings are fetched once; no update tracking or expired-listing detection
- Multi-user / multi-intent support — single user, single intent; multi-user is a future concern that the extensibility target feature makes easier
- LLM for the pre-scrape relevance gate — the gate stays heuristic to remain cheap; LLM is used only post-scrape on ingest, where the expensive work is already done
