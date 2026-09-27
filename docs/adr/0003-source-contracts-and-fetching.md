# ADR 0003 — Source contracts and fetching

- **Classify before routing.** `detect.Detect(url)` is a pure function that labels a URL as a known ATS, an Aggregator, or unknown HTML. Aggregator URLs are rewritten to their underlying ATS where possible, so the job comes from the cheap ATS API rather than the hostile aggregator. Adding an ATS means a `Detect` case plus a source, kept in step by table tests.
- **Capabilities are types, not flags.** Every source fetches pages. Fetching details is a separate, optional interface that only HTML/discovery sources implement; ATS sources return complete jobs from the listing.
- **Fetching is a dependency, not a source variant.** Sources parse and paginate; the transport is injected, so no parser knows about proxies.

## Bright Data Web Unlocker for protected sources

Hostile sources (LinkedIn, Indeed) fetch through Bright Data Web Unlocker; everything else (ATS APIs, cooperative boards) goes direct. Web Unlocker handles IP rotation, fingerprinting and CAPTCHAs and bills only successful responses. Datacenter IPs are pre-flagged on the aggregators, so there is no datacenter tier.

- Proxy use is a static property of each source, not user config.
- The worker requires a valid `BRIGHTDATA_PROXY_URL` at startup. A failed proxied fetch fails visibly and **never falls back to direct**, since a silent fallback hides that the wrong access path was used.
- Bright Data's zone spending limit is the only budget guard. When the zone reports exhaustion, proxied sources pause (direct sources keep running), a probe runs once a day, and they resume when it succeeds. Short-lived rate limits are ordinary retryable failures.
