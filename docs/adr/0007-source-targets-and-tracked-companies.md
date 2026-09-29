# ADR 0007 — Source Targets and tracked Companies

What to scrape is per-user data, not process-wide env vars. Sources are a fixed code registry; the worker scrapes the union of every User's interest into the shared `jobs` catalog.

## Source Targets (discovery)

Users choose Targets within a Source in `source_targets`. Each registry source has two independent attributes:

- **kind** is the shape of `value` and drives the input widget and validation: `board` is a token, `url` is a URL, and `filter` is a keyword plus structured `filters` (JSONB) declared per source.
- **role** is purpose: `ats` sources are tracked per Company (below); `discovery` covers aggregators and HTML boards. ATS sources are not valid Source Targets: `POST /source-targets` rejects them, and tracking is managed from Settings → Searches. An ATS URL pasted into a discovery source is rejected with a pointer to Company boards. The finer aggregator-vs-HTML split stays in `detect` (ADR 0003).

Creating a discovery Target starts a run immediately through the source's queue (ADR 0006).

## Companies (ATS)

`companies` is a shared catalog, like `jobs`, upserted on ingest, harvested (YC/Getro) or added from a pasted board URL. Users track a **Company**, not a board token, so interest survives a missing, additional or migrated Board.

- `tracked_companies` holds per-user interest and one Check Frequency per Company. A User can track a Company before any Board is known.
- A Company can have several Boards (`company_boards`). Only **verified** Boards are polled. Verification needs a successful fetch plus evidence of association (a link from the careers site or user confirmation); an ATS-shaped URL alone is not enough.
- Careers sites are inspected only for tracked Companies: when tracking starts, then about monthly, and sooner after repeated Board failures.
- Shared polling state (`board_poll_state`) is separate from interest. Each Board is polled once at the shortest interval any tracker asked for, and one worker holds each Board at a time.
- A replaced Board keeps being polled until two complete, empty checks close its remaining Jobs, then it is retired.
- The same vacancy on two ATS providers stays two Jobs; cross-provider merging is deferred.
