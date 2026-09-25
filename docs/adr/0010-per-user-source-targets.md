# ADR 0010 — Per-user Source Targets

What to scrape is per-user data in `source_targets`, not process-wide env vars. Sources themselves are a fixed code registry; users choose Targets within them. The worker scrapes the union of all users' enabled Targets into the shared `jobs` catalog.

Each registry source has two independent attributes:

- **kind** is the shape of `value` and drives the input widget and validation. `board` is a token, `url` is a URL, and `filter` is a keyword plus structured `filters` (JSONB) declared per source.
- **role** is purpose. `ats` means known-company boards, now tracked per Company (ADR 0016). `discovery` means search surfaces such as aggregators and HTML boards. Role drives UI grouping and guards: an ATS URL pasted into a discovery source is rejected with a pointer to Companies. The finer aggregator-vs-HTML split stays in `detect` (ADR 0004).

Creating a discovery Target starts a run immediately through the source's queue (ADR 0008) instead of waiting for the next tick.
