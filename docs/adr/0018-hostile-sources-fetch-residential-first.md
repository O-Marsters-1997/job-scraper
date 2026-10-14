# ADR 0018 — Hostile Sources fetch through residential proxies

ADR 0003 covers source classification and capabilities; this ADR covers how Hostile Sources are fetched.

## Context

LinkedIn is the most valuable discovery Source, and a hosted unblocker (Bright Data Web Unlocker) charges
$1.50 per 1,000 requests. About 91% of those requests are detail fetches, one per Job, so the price per
request sets the bill. Measured on 2026-10-01:

- LinkedIn's `jobs-guest` endpoints are public by design but rate-limited per IP. From a home IP, 41 search
  pages at 1 req/s drew no 429. The worker runs on Hetzner, whose datacenter IPs these sites flag, so
  going direct is not an option.
- Logged-out detail pages never carry `applyUrl` (0 of 40). Unlocker is logged out too, so it buys nothing
  a residential IP doesn't.
- `jobs-guest/jobs/api/jobPosting/{id}` is about 8KB gzipped, against 30KB for `/jobs/view/{id}`.
- At Decodo's $4/GB, a detailed Job costs about 18KB including retries and TLS setup, so roughly $0.07 per
  1,000 Jobs against $1.43–1.65 on Unlocker. The TLS figure is estimated, not measured.
- Bright Data's own residential zone has required company KYC since July 2026, so it is ruled out.

## Decision

- **Each Source declares a route, `Direct` or `Residential`** (`proxy.Route`). LinkedIn and Indeed are `Residential`; ATS APIs and cooperative boards stay `Direct`. Routes are a
  static property of the Source, never user config.
- **`Residential` sends every request through Decodo.** `DECODO_PROXY_URL` must be valid or the worker
  refuses to start while a `Residential` Source is registered.
- **A block is retried on fresh IPs, then fails.** A 429, 999, 403, or a redirect to LinkedIn's authwall or
  login counts as a block. The request is tried up to 3 times, each rotating the session ID and so the IP.
  If every attempt is blocked, the fetch returns `proxy.ErrBlocked` and the task fails and retries through the
  queue (ADR 0006).
- **One sticky Decodo session and keep-alive connection serves all requests, and rotates on a block.** A
  connection error rotates it too, since Decodo sessions expire. Spike #536 measured a `jobPosting` fetch at
  18.5KB billed with a new connection and session each time, of which about 5KB was CONNECT and TLS. One
  reused connection on one session cost 12.85KB (30.6% less), with 0 blocks in 15. A 404 or 410 means the Job
  is gone and is never retried. Each session owns a cookie jar that rotation discards, and rotation leaves
  other sessions' warm connections open; a blocked session's idle connections age out through `IdleConnTimeout`
  (#967).
- **The route never falls back to direct.** 0003 banned silent fallback because it hides that the wrong
  route was used. Requests are counted per route and outcome:
  `jobscraper_fetch_requests_total{source,route,outcome}` and `jobscraper_fetch_bytes_total{source,route}` (decoded body bytes) and
  `jobscraper_fetch_wire_bytes_total{source,route,direction}` (proxy connection bytes, comparable with
  Decodo's billing).
- **The fetch cache (ADR 0016) wraps the residential route.** A cached 200 skips the proxy.
- **Spend is capped in Decodo's own dashboard.** There is no zone-exhaustion gate.

Rejected:

- **Unlocker as the primary or fallback tier:** about 20× the cost per Job, and a second vendor, credential
  and budget guard to keep valid.
- **Scraping APIs (Scrapingdog $0.50–1.10/1K) and Apify actors ($0.37–1.00/1K):** cost more and add a
  third-party parser that can break.
- **LinkedIn alert emails:** they replace only search pages, about 9% of requests.

## Consequences

- We handle LinkedIn's blocks ourselves. Every retry costs bandwidth, which is cheap.
- One proxy vendor and one credential. A dead Decodo zone shows up as `ErrBlocked` failures and a rising
  error rate, and there is no safety net if LinkedIn hardens against residential IPs.
- The cost estimate depends on the small `jobPosting` endpoint. Going back to `/jobs/view` would cost about 4× in
  bandwidth.
- Plan: `plans/cheaper-hostile-sources.md`.
