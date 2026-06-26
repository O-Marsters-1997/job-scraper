# ADR 0014 — BrightData Web Unlocker replaces proxy tiers

**Status:** Accepted  
**Supersedes:** ADR 0009

## Context

ADR 0009 established a three-tier proxy selector (direct / datacenter / residential) with two env vars (`PROXY_DATACENTER_URL`, `PROXY_RESIDENTIAL_URL`). In practice only residential-grade unblocking is ever used: datacenter IPs are pre-flagged on LinkedIn/Indeed, and the ATS sources go direct. The three-tier enum existed to leave the door open for the datacenter tier but it was never wired to a real provider — the env var was always empty and the tier silently fell back to direct.

The decision now is to wire a real unblocking provider — BrightData Web Unlocker — and simplify the model to match the actual two-state reality.

**Why Web Unlocker over raw residential proxies:**  
Web Unlocker is a managed, pass-through proxy (`brd.superproxy.io:33335`) that handles IP rotation, browser fingerprinting, CAPTCHA solving, and JS rendering automatically. You pay per *successful* response (~$1.5–3 / 1k), so failed/blocked attempts don't cost anything. This is preferable to maintaining the retry/backoff/fingerprinting logic ourselves on top of raw residential IPs.

**Why not the BrightData managed LinkedIn Jobs scraper** (dataset `gd_lpfll7v5hcqtkxl6l`): that product has a completely different shape (async `POST /trigger` → poll snapshot → structured JSON). It's not a proxy; it doesn't fit the existing `Source.Iterate` / `PaginatedBase.Get` flow. Deferred as a separate integration if/when maintaining the hand-rolled LinkedIn parser becomes the pain point.

## Decision

Replace the three-tier enum with a single `UseProxy bool` on `sources.Config`. When true, `internal/proxy.Transport` builds an `http.Transport` routing through `BRIGHTDATA_PROXY_URL`. When false (zero value), it returns `http.DefaultTransport`.

Sources that previously used `proxy.Residential` (`linkedin`, `indeed`) set `UseProxy: true`. `wis` (previously `proxy.Datacenter`, which was always falling back to direct anyway) is now explicitly direct — it's a cooperative HTML board, not a hostile aggregator. All ATS sources keep the default (false → direct).

The env var is `BRIGHTDATA_PROXY_URL` — a full proxy URL including credentials, e.g. `http://brd-customer-<id>-zone-<zone>:<pass>@brd.superproxy.io:33335`. If unset, proxied sources degrade to direct rather than crashing (same graceful-fallback behaviour as ADR 0009).

`TLSClientConfig: &tls.Config{InsecureSkipVerify: true}` is set on the proxy transport only, because Web Unlocker performs TLS interception (MITM) to execute JS and handle CAPTCHAs. The hop to `brd.superproxy.io` is authenticated by the zone password, and the data fetched is public job listings. Upgrade path: add BrightData's CA certificate to a custom `x509.CertPool` and set `TLSClientConfig.RootCAs`.

## Consequences

- The datacenter tier is removed. If a source is ever found to work on datacenter IPs (cheaper), the upgrade path is either a second `UseProxyDC bool` field or a separate zone URL — straightforward once the need is empirically confirmed.
- Per-source proxy opt-in is now a single bool, static in code. No DB column, no per-user toggle. Proxy use is a property of a source's anti-bot requirements, not a user preference.
- `PROXY_DATACENTER_URL` and `PROXY_RESIDENTIAL_URL` env vars are removed; replace with `BRIGHTDATA_PROXY_URL` in `.env`.
- BrightData Web Unlocker is now a named infrastructure dependency. Cost scales with the number of proxied source fetches; at personal-use volumes this is negligible.
