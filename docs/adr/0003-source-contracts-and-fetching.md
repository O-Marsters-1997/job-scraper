# ADR 0003 — Source contracts and fetching

- **Classify before routing.** `detect.Detect(url)` is a pure function that labels a URL as a known ATS, an Aggregator, or unknown HTML. Aggregator URLs are rewritten to their underlying ATS where possible, so the job comes from the cheap ATS API rather than the hostile aggregator. Adding an ATS means a `Detect` case plus a source, kept in step by table tests.
- **Capabilities are types, not flags.** Every source fetches pages. Fetching details is a separate, optional interface that only HTML/discovery sources implement; ATS sources return complete jobs from the listing.
- **Fetching is a dependency, not a source variant.** Sources parse and paginate; the transport is injected, so no parser knows about proxies.

## Fetch routing

Hostile sources (LinkedIn, Indeed) fetch through Decodo residential proxies; everything else (ATS APIs, cooperative boards) goes direct (ADR 0018). Datacenter IPs are pre-flagged on the aggregators, so there is no datacenter tier.

- Proxy use is a static property of each source, not user config.
- The worker requires a valid `DECODO_PROXY_URL` at startup when a residential source is registered. A failed proxied fetch fails visibly and **never falls back to direct**, since a silent fallback hides that the wrong access path was used.
- Short-lived rate limits are ordinary retryable failures.
