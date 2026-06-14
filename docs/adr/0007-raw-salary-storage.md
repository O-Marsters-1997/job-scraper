# ADR 0007 — Store salary raw, defer numeric normalisation

**Status:** Accepted

## Context

Sources report salary inconsistently: ranges, single figures, currencies, hourly vs annual, and free text like "competitive". The email digest's `Remuneration` field is currently always empty because no salary is stored at all. Three options were considered:

- **Raw string only** — store exactly what the Source gave.
- **Raw + LLM-normalised** — also emit `salary_min_gbp`/`salary_max_gbp` from the Haiku ingest call.
- **Raw + hand-built parser** — deterministic Go normalisation across formats.

Normalised numeric columns would enable cross-source salary sorting/filtering, but that capability is not needed for v1, and both normalisation routes carry cost: the parser is ongoing maintenance across Source formats; the LLM route ties salary accuracy to model output and complicates the scoring call's contract.

## Decision

Store salary **raw only**: a single `salary_raw TEXT` column on `jobs` (a genuine property of the shared Job, so it belongs on `jobs`, not `job_scores`). This immediately populates the digest `Remuneration` field. The suitability rubric reads `salary_raw` as text — the LLM already reasons over free text, so it needs no pre-parsed numbers to factor salary into the fit score.

Numeric normalisation (min/max GBP) and salary-based numeric filtering are **out of scope for v1**.

## Consequences

- No cross-source numeric salary sort/filter in v1.
- Adding normalisation later is additive (new nullable columns + a backfill), not a reversal — but it *is* a future migration, accepted knowingly.
- `toJobData` populates `Remuneration` from `salary_raw`; the email templates already render it conditionally.
