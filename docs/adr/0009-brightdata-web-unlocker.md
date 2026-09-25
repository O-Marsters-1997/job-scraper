# ADR 0009 — Bright Data Web Unlocker for protected sources, no direct fallback

Hostile sources (LinkedIn, Indeed) fetch through Bright Data Web Unlocker. Everything else (ATS APIs, cooperative boards) goes direct. Web Unlocker handles IP rotation, fingerprinting and CAPTCHAs and bills only successful responses, which beats maintaining raw residential proxies. Datacenter IPs are pre-flagged on the aggregators, so there is no datacenter tier.

- Proxy use is a static property of each source, not user config. A source never pays for the proxy just because credentials exist.
- The worker requires a valid `BRIGHTDATA_PROXY_URL` at startup. A failed proxied fetch fails that check visibly and **never falls back to direct**, since silent fallback hides that the wrong access path was used.
- Bright Data's zone spending limit is the only budget guard. When the zone reports it is exhausted, proxied sources pause (direct sources keep running), a probe runs once a day, and the sources resume when the probe succeeds. Short-lived rate limits are ordinary retryable failures.
