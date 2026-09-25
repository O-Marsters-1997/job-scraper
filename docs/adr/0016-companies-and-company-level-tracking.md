# ADR 0016 — Companies catalog and company-level tracking

`companies` is a shared catalog, like `jobs`. Rows are upserted on ingest, harvested (YC/Getro) or added from a pasted board URL. Users track a **Company**, not a board token: interest survives a missing, additional or migrated ATS Board.

- `tracked_companies` holds per-user interest and one Check Frequency per Company. A User can track a Company before any Board is known.
- A Company can have several Boards (`company_boards`). Only **verified** Boards are polled. Verification needs a successful fetch plus evidence of association (a link from the careers site or user confirmation); an ATS-shaped URL alone is not enough.
- Careers sites are inspected only for tracked Companies: when tracking starts, then about monthly, and sooner after repeated Board failures. Untracked catalog entries are never crawled.
- Shared polling state (`board_poll_state`) is separate from user interest. Each Board is polled once at the shortest interval any tracker asked for, results are shared, and one worker holds each Board at a time.
- A replaced Board keeps being polled until two complete, empty checks close its remaining Jobs, then it is retired.
- The same vacancy on two ATS providers stays two Jobs; cross-provider merging is deferred.
