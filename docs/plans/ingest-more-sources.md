# Plan: Ingest more sources — a self-expanding aggregation engine

_Draft. A staged plan to grow coverage: finish the cheap public tier, then make the engine
discover and pull new companies on its own._

## Goal

Turn the scraper into a pipeline where a **discovery** process fills a shared catalogue of
companies and their job boards, and an **extractor** drains it — public JSON APIs first, proxies
only where forced. The engine should continuously find new companies (VC portfolios, aggregators,
domain crawls) that become cheap Tier-1 pulls on the next run.

## Where we are today

- **ATS extraction is done.** Greenhouse, Lever, Ashby, Workable, Recruitee and Personio all run
  through the shared `BoardSource`/`BoardSpec` (`internal/sources/board.go`), each hitting its public
  JSON/XML board API and yielding fully-populated `dto.Job`s with no enrich phase.
- **LinkedIn guest works but is thin.** `internal/sources/linkedin/linkedin.go` hits the
  `seeMoreJobPostings` guest endpoint but emits only `keywords`, `location`, and `start`.
- **Indeed is a stub** (`Iterate` logs and returns nil). Workday and SmartRecruiters don't exist.
- **Watched sources are per-user** (`source_targets`, ADR 0010). There is **no shared company
  catalogue** yet — that's the in-flight `companies` table this plan builds on.
- **Routing exists:** `detect.Detect(url)` classifies a URL to its ATS by host; `detect.RewriteToATS`
  turns an aggregator link into its underlying ATS URL before enqueue; `detect.ResolveBoard` turns a
  pasted board URL into `{source, token}`.
- **The pipeline shape is right:** the ATS-vs-HTML branch is decided by whether a source implements
  `DetailFetcher` (ADR 0011). Sources that yield full `dto.Job`s and skip `DetailFetcher` flow
  through the cheap ATS path — score, then `BulkExport`, no queue, no enrich.
- **Dedup is URL-only** (`jobs.url UNIQUE`, `NewURLs`, Valkey `ZADD NX`); scrape-once holds.
- **Proxy is a single switch:** `Config.UseProxy` routes through BrightData Web Unlocker
  (ADR 0014). No proxy tiers.
- **No robots.txt / crawl-delay honouring anywhere**, and no HTML-body ATS fingerprinting — both
  needed for the discovery crawl.

## Principles

1. **Registry = the global `companies` table.** Discovery writes companies and their boards there;
   the ATS path drains it. Shared and global, additive to per-user `source_targets`.
2. **Cheapest viable path first.** Public API → direct HTML → proxy, decided per source.
3. **Change-detection only on cheap, non-proxy sources.** Worthwhile for continuous ATS ingestion
   (free, re-polled); never for a proxy-backed source — the value doesn't justify the cost.
   Scrape-once stays the guarantee everywhere else.
4. **New feed sources need no new plumbing.** A source whose `Iterate` returns full `dto.Job`s and
   doesn't implement `DetailFetcher` automatically takes the cheap ATS path.

---

## Stage 1 — Finish the cheap public tier (no new infra)

### 1a. LinkedIn filter port

Extend `linkedin.Search` and `Search.pageURL` (`linkedin.go:54-67`) to emit the guest query params
we're missing, and declare them as `FilterField`s on the registry entry (`registry.go:74`):

| Filter | LinkedIn param | Notes |
|---|---|---|
| Company targeting | `f_C={id}` | **The registry hook** — pull all roles for a known company by LinkedIn ID. Needs a `linkedin_company_id` on the `companies` table. |
| Recency | `f_TPR=r{seconds}` | Incremental polling (past-24h); already URL-deduped. |
| Remote/hybrid/on-site | `f_WT` | 1 on-site / 2 remote / 3 hybrid. |
| Experience | `f_E` | |
| Job type | `f_JT` | F/P/C/I/T. |
| Distance | `f_D` + `geoId` | `location` alone can't express distance. |
| Salary band | `f_SB2` | |

Encode the caveats as comments: **`easy_apply` no longer works** (drop it), and LinkedIn **429s
after ~page 10 per IP** — mitigated by `UseProxy: true`; cap pages. Keep the guest
search-then-`GetDetails` flow; the detail parse already pulls description/salary/apply-URL. Adding
`job_level` + `company_industry` from the detail page needs new `dto.Job` fields — defer unless wanted.

**Files:** `linkedin.go`, `registry.go`, `builder/build.go` (map `t.Filters` → Search fields),
`linkedin_test.go`.

### 1b. New JSON-feed sources — RemoteOK, Remotive, HN "Who is Hiring", YC

Global feeds returning full content → model as **`Iterate`-yields-full-`dto.Job`, no
`DetailFetcher`** so they flow through the ATS path (scored, exported, no enrich). Register as
`role=discovery`. For `kind`: reuse `kindFilter` with a keyword `value` (these APIs have no
server-side search, so we filter client-side) or add a `kindFeed` if the add-target UX shouldn't
demand a keyword — **decide during tech planning; `kindFilter` is the lazy default.**

- **RemoteOK** — `https://remoteok.com/api` (JSON array, full descriptions). No proxy.
- **Remotive** — `https://remotive.com/api/remote-jobs` (JSON, full descriptions). No proxy.
- **HN Who-is-Hiring** — Algolia `hn.algolia.com/api/v1/search?tags=comment,story_{id}`. Freeform
  comments, low structure — lowest data quality, consider deferring.
- **YC** — `api.ycombinator.com/v0.1/companies` feeds the **companies table** (a Stage-2 harvester,
  not a job feed); `workatastartup.com` is a heavier jobs source, defer.

**Files per feed:** `internal/sources/{name}/{name}.go` + test + snapshot, `registry.go` entry,
`builder/build.go` branch, and a `detect.Detect` case only if their URLs surface in discovery.

### 1c. Indeed (stub → real)

Implement `Iterate`/`GetDetails` with `UseProxy: true`. Since it needs a proxy it can also live in
Stage 4; either way, validate 200 = real data (anti-bot pages return 200s).

---

## Stage 2 — Aggregators / pre-aggregated portfolios (harvest → companies table)

Harvest **company + board lists**, not jobs, and upsert them into `companies`:

- **Getro** (powers 850+ VC funds; `*.getro.co` / custom domains) — SPA over a JSON API; a
  base64-encoded filter param is the tell.
- **Consider**, **Wellfound**, **YC WaaS**, YC companies API (1b) — same pattern.

These are **harvesters**, a new component type: they yield company records
`{name, domain, linkedin_company_id?, ats?, token?}`, not `dto.Job`s, on their own schedule. New
package (e.g. `internal/discover/`). Depends on the companies table.

---

## Stage 3 — Discovery crawl (self-expanding)

For any `companies` row with a domain but no known jobs source:

1. **Seed** company lists — VC `/portfolio` pages, OpenVC, YC (Crunchbase is a paid API now, skip).
2. **Crawl the domain** for a careers page: `/careers`, `/jobs`, "we're hiring", `sitemap.xml`,
   `robots.txt`. **Honour robots.txt + Crawl-delay** (net-new — see Cross-cutting).
3. **Fingerprint the ATS** by URL *and HTML* signature (`boards.greenhouse.io`, `jobs.lever.co`,
   `jobs.ashbyhq.com`, `*.myworkdayjobs.com`, embedded ATS `<script>`/iframe markers). Extend
   `detect` with an HTML-body sniffer alongside the existing host matcher.
4. **Write back** `(company, domain, ats, token)` to `companies`. That row is then auto-drained by
   the Stage-1 ATS path next run — and, with `f_C` + a stored `linkedin_company_id`, becomes a
   targeted LinkedIn pull too.

This is the engine's compounding loop: Stages 2–3 continuously mint cheap Stage-1 pulls. New
package(s) for the careers crawler + ATS fingerprinter; reuses `detect` and the companies table.

---

## Stage 4 — Hard sources last (proxy required)

Only for coverage the ATS layer misses: **Indeed** (1c), **Glassdoor**, **LinkedIn-full**.

- Route via `UseProxy: true` (BrightData Web Unlocker).
- **Validate HTTP 200 = real data** — Cloudflare AI Labyrinth / anti-bot serve fake 200 pages; add
  a semantic validator, not just a status check.
- **Never authenticate our own LinkedIn account.** Guest only.

---

## Stage 5 — Normalize, dedup, schedule

- **Schema:** `dto.Job` is the single schema (already the case).
- **Dedup:** keep **URL-only** + `RewriteToATS`, which already collapses aggregator/ATS duplicates.
  Accept the minor residual dup where a feed URL and an ATS URL point at the same role
  (`// ponytail:` — add a `{ats}:{company}:{job_id}` key only if the dup rate is measured and
  material).
- **Change-detection (ATS tier only):** per ATS board, diff this run's URL set against the last;
  roles that disappear → mark closed. Needs a `jobs.status`/`closed_at` column and a per-board
  last-run URL set (Valkey or a `board_snapshots` row). **Gate strictly on
  `SourceRole == RoleATS && !UseProxy`** so it never touches proxy-backed sources. Emit new/closed
  deltas per run.
- **Scheduler:** cron + `MinScrapeInterval` already exist; add per-source schedules for the
  harvesters and crawl.

---

## Cross-cutting

- **robots.txt + crawl-delay:** net-new small module, required by Stage 3 and good citizenship for
  any HTML crawl. Identifying User-Agent is already set; add per-host delay from `Crawl-delay`
  (Lever declares `1`) and per-host rate limiting.
- **Companies ↔ worker wiring:** the worker scrapes the union of per-user `source_targets` today.
  Global companies must also be scraped — build the source set from
  `(enabled per-user targets) ∪ (companies with a known ats + token)`. This is what makes a
  discovered company a free pull for everyone.
- **Feed source kind:** reuse `kindFilter` vs add `kindFeed` (see 1b).

## Dependency: the `companies` table

The `companies` table is in flight. Before Stages 2–3 start, pin its schema. Discovery needs at least:

- Upsert-by-domain; columns for `name`, `domain`, `ats` (nullable), `token` (nullable),
  `linkedin_company_id` (nullable, for `f_C`), `last_seen`/`last_crawled`.
- Global (system-owned) ownership, not per-user.
- A query the worker uses to build the "companies with a known board" source set.

Define a thin `CompanyProvider` interface so Stage 2/3 code depends on the interface, not the
concrete migration — this work can then proceed in parallel with the table itself.

## Suggested build order

1. **Stage 1a** LinkedIn filters (no infra, immediate coverage). `f_C` ships disabled until the
   companies table carries `linkedin_company_id`.
2. **Stage 1b** RemoteOK + Remotive (cleanest feeds; HN / YC-jobs deferred).
3. **Companies table** reconciliation + `CompanyProvider` + worker union wiring.
4. **Stage 5** change-detection on the ATS tier (small, high-signal; unblocks "what's new/closed").
5. **Stage 2** Getro / Wellfound / YC harvesters → companies.
6. **Stage 3** careers crawl + HTML ATS fingerprint + robots politeness → companies writeback.
7. **Stage 4** Indeed, then Glassdoor / LinkedIn-full (proxy, semantic 200 validation).

## Verification (per increment)

```bash
go build ./... && go test ./...
go run ./cmd/snapshot rebase <source>   # for any new HTML/JSON parser
```

New sources ship with a snapshot test (real API/HTML fixture → expected `dto.Job`s), matching the
existing pattern in each `internal/sources/*/` package.
