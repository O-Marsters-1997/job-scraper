# ADR 0025 — The API polls third-party usage and holds it in memory

Issue: [#833](https://github.com/O-Marsters-1997/job-scraper/issues/833)

## Context

Admins want to see how close the Decodo proxy plan is to running out without opening Grafana or the
Decodo dashboard. Decodo exposes the figures over an authenticated HTTP API. ADR 0009 puts outbound
fetches in the worker and persistence in the API. ADR 0010 sources observability state from the
database and events from logs.

## Decision

- **The API calls Decodo itself.** `jobsearch` runs `GET https://api.decodo.com/v2/subscriptions` with
  `DECODO_API_KEY` on a 15-minute `schedule.Every`, started from `cmd/api/main.go`. This departs from
  ADR 0009: the call is a quota read for a page the API serves, not a scrape, and routing it through the
  queue and the worker would add a table or message for one number.
- **The latest snapshot lives in memory**, behind a mutex, and is not persisted. This departs from
  ADR 0010: the figure is a display value, not state worth alerting on. After a restart the snapshot
  refills on the first run, which `schedule.Every` starts immediately.
- **Failures keep the last good numbers.** A failed fetch sets `Status = error` with the message and
  leaves the last `FetchedAt`, and logs at warn with `provider=decodo`.
- **An unset key means no calls**, and the row reads `not_configured`.
- **`GET /usage/proxies` is admin-only** (`handlers.RequireAdmin`) and returns `{"providers": [QuotaView]}`
  using the shared `dto.QuotaView` and level function.
- **Units and auth are unconfirmed until checked with a real key.** The client sends
  `Authorization: Basic <key>` and treats `traffic_per_period` and `traffic_limit` as GB strings.

## Consequences

- The API now makes one outbound call to a third party on a timer, and `DECODO_API_KEY` joins its
  environment.
- A second API replica would poll separately; at one replica this is fine.
- A new provider means another `QuotaFetcher` and another entry in the `providers` list.
