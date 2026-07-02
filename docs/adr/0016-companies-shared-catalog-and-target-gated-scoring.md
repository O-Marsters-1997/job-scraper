# ADR 0016 — Companies as a shared catalog; suitability scoring gated by Source Targets; per-target check frequency

**Status:** Accepted

## Context

ADR 0015 gave ATS-role **Source Targets** an implicit "tracked company" meaning, but
there was no first-class Company record — only the `company_slug` string carried on
each Job. Two capabilities were missing:

1. A catalog of every company the scraper has ever encountered (not just ones a user
   chose to track), so users can browse and toggle tracking rather than needing to
   already know a company's ATS.
2. A way to re-check a tracked company's board on a cadence the user chooses, rather
   than the single global 6-hour tick / 5-hour min-interval gate shared by every ATS
   platform.

Suitability scoring also fanned out to every user with an AI credential, regardless of
whether that user had any search or tracked company that would have surfaced the job.
This wasted LLM calls and, more importantly, gave users suitability scores for jobs at
companies they never asked to track.

## Decision

**Companies is a shared catalog, not a per-user table.** One `companies` row per
company `slug`, visible to all users — mirroring how `jobs` is a shared catalog while
tracking/scoring/applications are per-user overlays. Rows are upserted automatically
during ingest whenever a job carries an unseen `company_slug`, and can be added
manually by resolving a pasted ATS board URL. `ats_source`/`ats_token` are nullable:
companies seen only via discovery sources (LinkedIn, Indeed, Work in Startups) have no
known board.

Ceiling: uniqueness is on `slug` alone. A company that runs boards on two ATSes, or two
distinct companies whose names slugify identically, collapse into one row. Upgrade path
if this bites: `UNIQUE(ats_source, ats_token)` plus a slug-alias table. Not built now —
`jobs.company_slug` is already slug-granular and this hasn't been observed as a problem.

**Tracking a Company reuses Source Targets — no new tracking table.** A "tracked
company" is exactly an enabled ATS-role Source Target whose `(source, value)` matches
the Company's `(ats_source, ats_token)`. Toggling tracking on a Company page
upserts/enables that Target (via `company_id` as an optional back-reference, set for
book-keeping); toggling off disables it. This was a deliberate choice over a parallel
`tracked_companies` table: a second table would give the worker two sources of truth
for what to scrape, and pre-existing ATS targets (created before this feature) would
need a backfill to appear as "tracked" — instead they light up for free via the
`(source, value)` join.

**Suitability scoring is gated by Source Targets, not "every user with a credential".**
For an ATS job, only users with an enabled Target for that exact `(source,
company_slug)` — i.e. users tracking that company — get it scored. For a discovery job
(no fixed company per user), users with *any* enabled Target on that source get it
scored, since discovery jobs aren't attributable to a single company per user. This is
a behaviour change from before: a user with an AI credential but no matching Target
now gets zero suitability scores for that job, where previously they got one
regardless of relevance to their search.

**Check frequency is per-Target, gated in SQL, not per-platform in Valkey.**
`source_targets` gains `check_interval_minutes` (default 360, minimum 60 enforced at
the API) and `last_checked_at`. The worker's reload tick moves from 6-hourly to
hourly (`sources.DefaultSchedule`); each tick loads only *due* targets
(`ListDueSourceTargets`, SQL `WHERE enabled AND (last_checked_at IS NULL OR stale)`)
rather than all enabled targets. `last_checked_at` is written per board token via a
`BoardSource.WithDone` hook, called after each token's page callback succeeds — not
after the whole board list, so one dead board doesn't block the timestamp for working
ones (this also fixed a pre-existing bug where the first failing token aborted every
remaining token on that ATS in one `Iterate` call). The orchestrator's existing
Valkey-based `MinScrapeInterval` gate is skipped entirely for ATS-role sources — the
SQL due-filter is the gate — but is kept for discovery sources, which have no
per-target freshness column and still need the platform-wide throttle now that the
tick runs hourly instead of every 6 hours. On-demand scrape-now requests still bypass
scheduling entirely (per ADR 0012) and never touch `last_checked_at`.

## Consequences

- Users get a "companies I've seen" catalog without having to already know a
  company's ATS, and a toggle-driven tracking UX, while the scraper still has exactly
  one source of truth (`source_targets`) for what to scrape.
- Suitability scoring cost drops for large user bases, since it no longer fans out to
  users with no interest signal in a given job's company/source. Users who relied on
  the old "score everything" behaviour without configuring any targets will see fewer
  scores — this is intended, not a regression.
- Per-company check frequency required moving the reload tick to hourly; this is safe
  for discovery sources only because the existing 5-hour Valkey gate remains in place
  for them.
- `BoardSource.Iterate` continuing past a failed token (rather than aborting the whole
  board) is a behavioural fix that benefits all six ATS sources, not just companies
  with configured frequencies.
