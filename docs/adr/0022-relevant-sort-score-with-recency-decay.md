# ADR 0022 — Relevant sort: Suitability decayed by age

Issue: [#788](https://github.com/O-Marsters-1997/job-scraper/issues/788)

## Context

The jobs list was ordered by Suitability DESC then `scraped_at` DESC, with a few wildcard slots injected
from lower-ranked jobs. Suitability alone buries recent jobs when scoring is off or the User's Picks are
thin. Recency alone surfaces jobs that do not fit. Wildcards were meant to counter the first problem but
were rarely useful and made the order unpredictable. The only way to re-sort was clicking a column header.

## Decision

- **Relevant is the default sort.** `rank = score x 0.5^(ageDays / 3)`: the score halves every three days.
  An 80 from three days ago ties a 40 from today; a 90 from today beats a 60 from today.
- **Age comes from `first_discovered_at`**, not `scraped_at`. A re-scrape or reopened Job does not look
  new. A future timestamp (clock skew) clamps to age 0, so it never boosts.
- **Unscored jobs use a prior of 50**, the middle of the scale, so they are neither hidden at the bottom
  nor ranked as Great.
- **Ties break by `first_discovered_at` DESC, then ID**, so the order is deterministic.
- **It runs in the client**, over the already-loaded unpaginated list. No server sort parameter; `GET
  /jobs/all` keeps plain server order (score DESC, `scraped_at` DESC). The constants (`HALF_LIFE_DAYS`,
  `UNSCORED_PRIOR`) sit beside the function in `frontend/src/lib/jobSort.ts`.
- **A sort control sits top-right of the table** with Relevant, Newest and Best match. The choice lives in
  the `sort` URL param; an absent or unknown value means Relevant. A header sort shows the control as
  "Custom".
- **Wildcards are removed**: slotting, the DTO and zod field, the badge and the default-view check.

## Alternatives

- **Hacker News gravity** (`score / (age + 2)^1.8`). Built for unbounded vote counts. Our score is capped
  at 100, and its decay is steep early and flat late, so a good job of two weeks falls off differently
  from one of two days with no knob that maps to "days".
- **Recency buckets** (today, this week, older, each sorted by score). Easy to explain but a job moves
  rank in a step at midnight, and the bucket edges are arbitrary.
- **Sort by Suitability only, or by recency only.** Both failure modes in Context.
- **A server-side sort parameter.** The list is already loaded whole, and a second implementation of the
  formula in SQL would have to be kept in step.
- **Keep wildcards beside Relevant.** They fight the formula: the decay already lets fresh unscored jobs
  surface.

## Consequences

- Changing the half-life or prior is a one-line edit with a check in `jobSort.check.ts`.
- The order shifts as jobs age, even with no new scrapes. Reloading later can reorder the list.
- Relevant depends on the whole list being loaded. If the list is ever paginated, the sort has to move to
  the server.
- Wildcard code and its tests are gone; nothing replaces the occasional low-ranked job surfacing.
