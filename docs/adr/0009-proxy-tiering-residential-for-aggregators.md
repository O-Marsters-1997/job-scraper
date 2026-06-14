# ADR 0009 — Per-source proxy tiering; residential from the start for aggregators

**Status:** Accepted

## Context

There is no proxy infrastructure today. Adding Aggregator discovery (LinkedIn/Indeed first) requires it, but routing every Source through residential proxies would be needlessly expensive. The guiding principle from the approach doc is "**proxy as last resort** — tier proxy grade to source hostility": API/friendly Sources go direct or via datacenter IPs; only hostile aggregators need residential.

That principle, applied literally, argues for starting even LinkedIn on datacenter + rate discipline and escalating to residential only if blocked — especially since Aggregators are used **discovery-only** (grab the URL, then extract from the underlying ATS) at personal low volume, which is a lower detection bar than profile scraping or content extraction.

Against that: the research is blunt that datacenter IP ranges (AWS/GCP/Azure) are pre-flagged on LinkedIn/Indeed, and aggressive LinkedIn scraping gets flagged within ~200 pulls, with account/legal risk attached. A flagged datacenter attempt risks the account before any coverage is gained.

## Decision

Build a **per-source proxy-tier selector** (direct / datacenter / residential) injected into the fetch transport, defaulted to direct and opt-in per Source by config — so proxy grade is enforced by configuration, not habit.

For the **Aggregator tier specifically, route through residential from the start** rather than starting datacenter and escalating. This is a deliberate override of "proxy as last resort": the block/account/legal risk of testing aggregators on pre-flagged datacenter IPs is judged too high to probe, so we pay the most expensive tier up front to minimise block risk on the one Source class that actively fights scraping.

## Consequences

- The "proxy as last resort" principle still governs every *other* Source (API direct, friendly HTML via datacenter); the override is scoped to Aggregators only.
- Residential proxy support is a **hard dependency** of the Aggregator discovery feature, not an optional enhancement — it must ship with it.
- Cost is incurred for residential bandwidth before its necessity is empirically confirmed; accepted as the price of not gambling the account. If discovery-only volume proves to survive datacenter, the tier is config-flippable down later without code change.
- The tier selector is config-driven and testable as a pure mapping (tier → transport), independent of live proxies.
