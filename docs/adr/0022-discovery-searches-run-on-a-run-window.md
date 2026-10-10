# ADR 0022 — Discovery searches run on a Run Window

Supersedes the rule in ADR 0007 and `CONTEXT.md` that a discovery Source Target runs only when added or
explicitly rerun. Decided in spike #407.

A saved LinkedIn search that only runs when someone clicks rerun misses the hours when early applicants win.
LinkedIn and WIS already fetch only what is new since the last success (`linkedin.Recency`, `wis.Recency`,
NewestFirst early-stop), and detail fetches skip known URLs through `NewURLs`, so a frequent run usually
costs one listing page.

## Decision

- **Each Source Target has its own Run Window.** It is an interval within chosen weekdays and hours, in an
  IANA timezone. A NULL interval means manual only. New Targets default to hourly, Mon–Fri, 08:00–18:00,
  Europe/London, editable at creation and later. `enabled` keeps its meaning (the system or the User has
  turned the Target off), separate from manual versus automatic. ATS Boards keep their Check Frequency.
- **Due-ness lives in Postgres as `next_run_at`** on `source_targets`, with a partial index on automatic
  enabled rows. Each advance sets it to now + interval ± 10% jitter, moved to the next window opening if
  that falls outside the window. The arithmetic is plain Go, unit-tested across DST changes.
- **The worker claims due Targets once a minute**, through `schedule.Every(ctx, "discovery", time.Minute,
  js.PublishDueTargets)`. A claim takes at most N Targets ordered by `next_run_at`, skips any Target whose
  run is `queued` or `running`, advances `next_run_at` and starts a run as `StartSourceTargetRun` does.
  After downtime each overdue Target runs once, and Recency widens its window to cover the gap.
- **A create or manual rerun resets the clock** to now + interval, so a manual run is not followed minutes
  later by a scheduled one. RecoverRuns still owns stuck runs.
- **A failed run keeps the interval.** Retries within a run stay with RabbitMQ (ADR 0006), and the fetch
  cache (ADR 0016) makes them free, so a broken Target costs about one paid listing page per run. There is
  no backoff and no auto-pause.
- **Disabling a Target alerts the operator.** `jobscraper_source_targets_disabled_total{source,reason}`
  feeds a Grafana alert next to the Error-burst and Dead-letter alerts. Users see `disabled_reason` on the
  Target in Settings.
- **Only incremental Sources can be automatic.** sourcespec declares an `Incremental` capability, and the
  API rejects an interval on any other Source. Indeed is not incremental until it sends `fromage` from
  LastSucceededAt with date sort and early-stop.
- **Each User may have at most N automatic Targets** (env, default 10). Manual Targets are unlimited.
- **Identical searches by two Users are two runs.** Jobs and detail fetches are already shared, and
  duplicated listing pages are about 9% of requests (ADR 0018).
- **Existing Targets stay manual** when this ships.

## Rejected

- **One global or per-Source interval:** Users can't make a low-priority search run less often.
- **Cron expressions:** too hard to build and explain in the UI, and too easy to make too frequent.
- **River or Oban periodic jobs:** they are global crons that don't persist, and Run Windows are per-User
  rows.
- **A shared search row fanned out to subscribers**, like `board_poll_state`: a schema split and a
  Candidate rewrite to save listing pages. Revisit when identical searches across Users are common.
- **Pausing inactive Users' searches:** cleaning up unused searches is the User's job.

## Consequences

- Hourly runs spend each owner's OpenRouter key on scoring all day, which is what makes ADR 0023 urgent.
- Proxy spend grows with automatic Targets × windows. The per-User cap and the Decodo traffic limit (ADR 0024) are
  the only bounds. There is no request budget.
- Cost per run is still to be measured from `jobscraper_fetch_requests_total` and
  `jobscraper_fetch_bytes_total` once on-demand discovery has been tested by hand (#407's blocker).
