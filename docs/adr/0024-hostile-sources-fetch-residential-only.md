# ADR 0024 — Hostile Sources fetch residential-only; Bright Data is dropped

Supersedes [ADR 0018](0018-hostile-sources-fetch-residential-first.md), and with it what was left of the
Bright Data section of ADR 0003. Amends the Bright Data specifics of
[ADR 0016](0016-fetch-cache-makes-proxied-retries-free.md); the Fetch Cache itself stands.

## Context

Since 0018, Decodo residential has carried LinkedIn, Indeed and WIS, and Bright Data Web Unlocker has
served only as the counted fallback after three blocked residential attempts. No Source routes through
Unlocker directly. Decodo is doing the job on its own, so the fallback costs a second vendor, a second
credential, a zone gate and a daily probe for a safety net we don't use.

## Decision

- **Bright Data is removed entirely.** The `unlocker` route, the tiered fallback, the Unlocker transport,
  `ZoneGate` and `client_10100` detection, the daily `geo.brdtest.com` probe, and `BRIGHTDATA_PROXY_URL` /
  `BRIGHTDATA_CA_CERT` all go. So does the Unlocker half of the Grafana proxy-spend panel.
- **A Source's route is `direct` or `residential`.** `residential` replaces `tiered`. LinkedIn, Indeed and
  WIS are `residential`; everything else stays `direct`. Routes stay a static property of the Source.
- **A block is retried on fresh IPs, then the fetch fails.** The block rules, the 3 attempts with a new
  session each, and the sticky-session rotation from 0018 are unchanged. After the third block the request
  fails like any other fetch error, and the task retries through the queue (ADR 0006).
- **Still no fallback to direct.** The 0003 rule stands: a failed proxied fetch fails visibly.
- **The worker requires a valid `DECODO_PROXY_URL`** at startup when a `residential` Source is registered.
- **Decodo's own traffic limit is the only budget guard.** The planned Third-party usage stats page shows
  how much of it is left; nothing detects exhaustion from proxy errors.
- **`jobscraper_fetch_fallbacks_total` and the 20% fallback alert are removed.** A dead or exhausted Decodo
  plan shows as `blocked`/`error` outcomes on `route="residential"` in `jobscraper_fetch_requests_total`.

Rejected:

- **Keep Unlocker as a dormant fallback:** it keeps a second vendor's credentials, code paths and alerts
  alive for a case that hasn't happened.

## Consequences

- No safety net if LinkedIn hardens against residential IPs, which is the risk 0018 kept Unlocker for. If it
  happens, those Sources fail visibly until a new route is chosen.
- One proxy vendor and one credential to keep valid.
- The Fetch Cache (0016) still saves Decodo bandwidth on retries, and its rules carry over to the residential
  route: cache a 200 or 3xx hop, never a 403 or 5xx, and treat 404/410 as gone. The Bright Data error
  header checks and the zone-probe exemption fall away.
