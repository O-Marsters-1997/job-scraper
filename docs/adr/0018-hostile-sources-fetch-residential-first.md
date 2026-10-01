# ADR 0018 — Hostile Sources fetch residential-first, with an Unlocker fallback

Supersedes the "Bright Data Web Unlocker for protected sources" section of ADR 0003. The rest of 0003 stands.

## Context

LinkedIn is the most valuable discovery Source, and every LinkedIn and Indeed request went through Web
Unlocker at $1.50 per 1,000. About 91% of those requests are detail fetches, one per Job, so the price per
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

- **Each Source declares a route, `direct` or `tiered`.** It replaces the `UseProxy` bool. LinkedIn and Indeed
  are `tiered`; ATS APIs and cooperative boards stay `direct`. Routes are still a static property of the
  Source, never user config.
- **`tiered` tries Decodo residential first.** `DECODO_PROXY_URL` sits alongside `BRIGHTDATA_PROXY_URL`, and the
  worker refuses to start if a `tiered` Source is registered and either is missing or invalid.
- **A block is retried on fresh IPs, then sent once through Unlocker.** A 429, 999, 403, or a redirect to
  LinkedIn's authwall or login counts as a block. The request is retried up to 3 times, each with a new
  session ID and so a new IP, then that one request goes through Unlocker. A 404 or 410 means the Job is
  gone on whichever route answered, and it never falls back.
- **Neither route ever falls back to direct.** 0003 banned silent fallback because it hides that the wrong
  route was used. This fallback stays on a paid proxy and is counted:
  `jobscraper_fetch_requests_total{source,route,outcome}`, `jobscraper_fetch_bytes_total{source,route}` and
  `jobscraper_fetch_fallbacks_total{source}`. An alert fires when more than 20% of residential requests
  fall back over an hour.
- **The fetch cache (ADR 0016) wraps both routes.** A cached 200 skips both, and a residential 200 is cached
  just as an Unlocker one is.
- **The Unlocker budget guard is unchanged.** The zone-exhaustion pause and daily probe from 0003 now apply only to
  the Unlocker route. Decodo has no zone gate; its spend is capped in Decodo's own dashboard.

Rejected:

- **Unlocker only:** about 20× the cost per Job.
- **Residential only:** no safety net if LinkedIn hardens against residential IPs.
- **Scraping APIs (Scrapingdog $0.50–1.10/1K) and Apify actors ($0.37–1.00/1K):** cost more and add a
  third-party parser that can break.
- **LinkedIn alert emails:** they replace only search pages, about 9% of requests.

## Consequences

- We handle LinkedIn's blocks ourselves on the residential route. Every retry costs bandwidth, which is
  cheap, rather than nothing, which is what an unbilled Unlocker failure costs.
- Two proxy vendors and two credentials to keep valid. A dead Decodo zone shows up as a fallback rate of
  100%, and the fallback alert catches it before Unlocker spend climbs.
- The cost estimate depends on the small `jobPosting` endpoint. Going back to `/jobs/view` would cost about 4× in
  bandwidth.
- Plan: `plans/cheaper-hostile-sources.md`.
