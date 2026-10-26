# ADR 0036 — Stored relevance key for the Relevant sort

Issue: [#1330](https://github.com/O-Marsters-1997/job-scraper/issues/1330). Status: proposed, design only.
Extends [ADR 0022](0022-relevant-sort-score-with-recency-decay.md). Nothing here is implemented.

## Context

`PageJobs` with `sort=relevant` evaluates `score x 0.5^(age/3d)` for every eligible job, sorts, then
takes one page. On the 1x seed 42 dataset that is about 19.5k eligible rows per user (the 90-day window
and `closed_at` cut the 82k jobs). Page 1 costs 50-155 ms locally for that sort alone.

## Decision

- **The sort key is time-invariant.** `rank = s x 2^(-(now - ts)/3d)`, so `log2(rank) = (log2(s) + ts/3d) - now/3d`.
  The last term is the same for every row in a query, so ordering by `rel_key = log2(s) + epoch(ts)/259200`
  gives the same order. `s = COALESCE(suitability_score, 50)`, `ts = COALESCE(posted_at, first_discovered_at)`.
  `s = 0` maps to `-Infinity`, which ties exactly as rank 0 does today.
- **`relevance_key(score int, ts timestamptz) float8`** is one `IMMUTABLE` SQL function. It converts with
  `ts AT TIME ZONE 'UTC'` because `extract(epoch FROM timestamptz)` is only `STABLE` and cannot be used
  in a generated column or an index.
- **`jobs.rel_key_unscored float8`**: the key for a job the user has no score row for (prior 50). A plain
  nullable column, not a generated one: `ADD COLUMN ... GENERATED ... STORED` rewrites `jobs` under
  `ACCESS EXCLUSIVE`. A `BEFORE INSERT OR UPDATE OF posted_at, first_discovered_at` trigger
  (`jobs_rel_key_unscored_trg`) keeps it current on ingest; a batched backfill fills existing rows.
- **`job_scores.rel_key float8`**, maintained by two triggers: `job_scores_rel_key_trg`, `BEFORE INSERT OR
  UPDATE OF suitability_score` (first score and every rescore, `UpsertJobScore` and
  `UpdateJobScoreBreakdown`), and `jobs_rel_key_scores_trg`, `AFTER UPDATE OF posted_at, first_discovered_at`
  on `jobs` (posted dates move on re-scrape through `LEAST(posted_at, ...)`). Both are proposed, pending the
  ADR 0011 question in Consequences. `job_scores_rel_key_trg` reads the job with `FOR SHARE`, which
  serialises it against a concurrent posted-date update, so a score insert cannot commit a key built
  from the old date after the fan-out has run.
- **A NULL score is not 0.** NULL takes the prior 50 and joins the scored stream; 0 maps to `-Infinity`.
- **Indexes:** `job_scores (user_id, rel_key DESC) WHERE NOT blocked` and
  `jobs (rel_key_unscored DESC, first_discovered_at DESC, id) WHERE closed_at IS NULL`.
- **`PageJobs` merges two streams** for `sort=relevant`: scored rows from `job_scores`, and jobs with no
  `job_scores` row for the user from `jobs`. Each stream carries the full filter set, orders by
  `key DESC, first_discovered_at DESC, id` and takes `LIMIT offset + page_limit`. The outer query orders
  the union the same way and applies `OFFSET`/`LIMIT`. Tie-breaks stay as ADR 0022 states.
- **Deep offsets fall back to the current sort.** Each stream reads `offset + limit` rows (`page_size` is
  clamped to 100, ADR 0031), so cost grows with the offset. Past a cap (proposed 1000 rows) the query keeps
  today's path. The cap is a proposal.
- **`newest` and `best` are unchanged.** `best` already has `job_scores_user_best_idx`.

## Evidence

On 1x seed 42, 22 users (20 seeded, the real user, one user with no scores) at two query clocks
(2026-10-26 and 2026-11-30), the full ordered ID list of the current formula, of `rel_key`, and of
`rel_key` with `ts` clamped to `first_discovered_at` have identical md5 and zero positional mismatches
(44 of 44). Pages at offsets 0, 50, 950, 5000 and 15000 of the merge query equal the current query for
every user (198 non-empty pages, 22 empty). The merge pages ran against a scratch copy with
`rel_key` in a partial index on the breakdown predicate in place of `NOT blocked`, which the dev DB
lacks. Timings are indicative only, taken by hand beside a busy query and not archived: page 1 of
the merge about 0.2 ms (no scores) to 3.5 ms (5,000 scores) against 49-155 ms for the current sort;
at offset 5000 the merge was slower (154 ms against 54 ms), which is why the cap exists. The
equivalence is an identity (`log2` of the rank minus a per-query constant); the seed check guards
float rounding and the tie-breaks.

## Consequences

- The Relevant formula now lives in `relevance_key`. Changing the half-life (259200 s) or the prior (50)
  is a migration that rewrites `job_scores.rel_key` and `jobs.rel_key_unscored`, not an edit to a query.
- A trigger on `jobs` writes `job_scores`, which ADR 0011 gives to scoring alone. Triggers are chosen
  because posted dates change in four jobsearch queries; the alternative is a scoring port called
  from each. Unresolved, listed as an open question on the issue PR.
- A `posted_at` in the future (clock skew) boosts a job until that time passes, where ADR 0022 clamps its
  age to 0; an `IMMUTABLE` function cannot read `now()`. This differs from ADR 0022 and is unproven, as
  the seed has none.
- The timings above are for the single-sort `PageJobs` from before #1369; re-time against the narrow
  `ranked` form when implementing.
