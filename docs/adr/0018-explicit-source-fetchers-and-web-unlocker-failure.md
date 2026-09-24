# ADR 0018 — Explicit source fetchers and Web Unlocker failure behavior

**Status:** Accepted; implementation pending
**Partially supersedes:** ADR 0014 (proxy configuration and fallback behavior)

## Context

ADR 0014 chose Bright Data Web Unlocker for sources that need managed unblocking and direct requests for cooperative ATS APIs and boards. Its `UseProxy` boolean chooses a transport in `NewBase`, but missing credentials and transport-construction errors silently fall back to direct requests. A protected source can therefore run through the wrong access path without an error. The user wants one source contract across providers while paying for Web Unlocker only where it is needed.

The current `Source` interface describes job enumeration, with optional `DetailFetcher` for details. It should remain independent of how HTTP content is acquired. The fetch variation belongs at the request boundary, so source adapters can use the same collection contracts with a direct fetcher or a Web Unlocker fetcher. Duplicating every source adapter into proxy and direct source types would spread parsing and pagination rules across both variants.

## Decision

- Keep the `Source` and optional `DetailFetcher` contracts. Give source adapters a fetch dependency with direct and Web Unlocker implementations; choose the implementation by source requirements. Cooperative ATS APIs and boards use direct access. Protected sources use Web Unlocker. A source never pays for Web Unlocker merely because the worker has credentials.
- Every scraper worker requires `BRIGHTDATA_PROXY_URL` at startup, even if its currently enabled sources all use direct access. Missing or malformed configuration prevents the worker from starting.
- A Web Unlocker fetch failure, including rejected or expired credentials, fails that source check visibly. It must not retry through direct access. Normal bounded retry of the same required access path may occur according to source policy.
- Bright Data's own zone spending limit is the spending guard; no separate application spending cap is planned. When Bright Data reports that the Web Unlocker zone has reached its usage limit, pause every source using the Web Unlocker fetcher. Direct sources, including ATS APIs, continue operating; paused sources do not fall back to direct access.
- While the zone is paused, run one scheduled Web Unlocker access probe per day. Resume the paused sources as soon as a probe confirms the zone works again. A failed probe leaves the sources paused and does not mark their source checks as successful. Short-lived rate limits remain ordinary fetch failures subject to bounded retry rather than suspending the whole zone.
- Web Unlocker is the only intended managed fetching product. No separate managed Web Scraper API or raw residential proxy integration is planned.

## Consequences

- The worker has a mandatory credential dependency even for direct-only source configurations. This is a deliberate startup contract in exchange for predictable deployment configuration.
- Source-specific routing controls paid requests and makes failures attributable to the correct access path. An invalid credential may only become observable on the first proxied request unless an explicit live preflight is introduced later.
- The zone spending limit has a fetcher-local blast radius. Protected-source work must retain enough state to resume without treating a budget pause as a successful check. Bright Data identifies an exhausted zone separately from temporary rate limits in its [proxy error catalog](https://docs.brightdata.com/proxy-networks/errorCatalog); its [zone usage limits](https://docs.brightdata.com/general/usage-monitoring/Usage) are assessed periodically rather than being exact instantaneous caps.
- The current `proxy.Transport` and `NewBase` fallback behavior must change. ADR 0014's Web Unlocker choice and direct-vs-protected distinction remain; its graceful direct fallback is superseded.
- No job parser needs to understand Bright Data's proxy URL or credentials. Access controls, timeouts, redirects, response-size limits and transport trust belong at the fetch boundary.
