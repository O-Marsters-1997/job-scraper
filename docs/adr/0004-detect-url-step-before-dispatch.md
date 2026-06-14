# ADR 0004 — Detect(url) classification step before dispatch

**Status:** Accepted

## Context

The scraper routes a discovered URL to a Source via `Dispatch`, which iterates Sources calling `CanHandle(url)` (prefix match) and lets the first match own extraction. Adding API-first extraction means a URL must be classified to its ATS platform so it can be pulled from a structured API instead of HTML-scraped. Aggregators (LinkedIn/Indeed) add a harder case: an aggregator-discovered URL must be **re-classified and rewritten to its underlying ATS** before any Source can own it — at discovery time there is no Source that handles a `linkedin.com/jobs/...` URL we actually want to *extract* from.

Two options:

- **Self-registration via `CanHandle`** — keep the existing seam; each Source declares the URLs it handles. Zero new abstraction, but aggregator→ATS rewriting has no home (no Source claims the aggregator URL for content), and classification logic is scattered across Sources.
- **A dedicated `Detect(url) ATSType` step** that runs before dispatch and yields the classification (a known ATS platform, an Aggregator, or unknown-HTML).

## Decision

Introduce a standalone **`Detect(url) ATSType`** classifier that runs before dispatch. Routing uses its result to pick the extraction method. Aggregator-classified URLs are sent back through `Detect` after their underlying ATS URL is recovered, so they end up extracted from the cheap Tier-1 source rather than the hostile aggregator.

`Detect` is a pure function over a URL — no I/O — so it is exhaustively table-driven testable in isolation (one of the deep modules this work targets).

## Consequences

- A classification registry now lives alongside `CanHandle`. `CanHandle` is retained for the final Source selection; `Detect` owns the *kind* decision. The duplication is intentional: `Detect` answers "what is this URL?" and `CanHandle` answers "which Source instance handles it?".
- `CONTEXT.md` gains **ATSType**, **ATS**, and **Aggregator** terms.
- The aggregator discovery feature depends on this step — it is the home for the aggregator→ATS rewrite.
- Adding a new ATS means adding a `Detect` case plus a Source; the two stay in sync (covered by the detector's table-driven tests).
