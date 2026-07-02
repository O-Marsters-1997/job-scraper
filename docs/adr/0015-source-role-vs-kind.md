# ADR 0015 — Explicit source `role` (ATS vs Discovery), distinct from `kind`

**Status:** Accepted

## Context

The source registry classified each source only by `sourceKind` — `board`, `url`,
or `filter` — which describes the *shape of the `value`* a user supplies (a board
token, a full URL, or a keyword string plus structured filters). Kind drives which
input widget the add-target form shows.

Kind does **not** describe a source's *purpose*, and the two axes don't line up. An
ATS board (Greenhouse, Ashby, …) is a cheap, known-company check that the engine
re-polls periodically via a public API (`atsPath`, no Enrich phase). An Aggregator
(LinkedIn, Indeed) or HTML board (Work in Startups) is a discovery surface you
*search* — paginated, enqueued, enriched (`htmlPath`). Yet `kindFilter` covers both
WIS (a content board) and LinkedIn (a discovery aggregator), so kind alone can't tell
the UI or a validator whether a target is a tracked company or a search.

This surfaced two concrete problems: the Searches settings page presented every
target in one flat list with no cue that ATS boards auto-refresh, and the board-token
input accepted a full URL with no validation (nothing stopped a user pasting an Ashby
URL where a bare token belongs).

## Decision

Add a `role` attribute to each registry entry and to the `SourceInfo` API payload,
orthogonal to `kind`. Two values:

- `ats` — the six ATS board sources (greenhouse, lever, ashby, workable, recruitee, personio).
- `discovery` — everything else (wis, linkedin, indeed).

`role` is a facet of the existing **Source Target** concept, not a new entity. The
finer Aggregator-vs-HTML-board distinction is **not** duplicated into the config role;
it already lives at runtime in `detect.ATSType` and the `DetailFetcher` capability
split (see ADR 0011). `role` exists purely to express purpose at the config/UX layer.

`role` drives:

- **UI grouping** — the Searches page splits targets into "Tracked companies" (ats)
  and "Discovery searches" (discovery).
- **Light guards** in the Create handler — an ATS board-token value that looks like a
  URL is rejected, and an ATS board URL pasted into a discovery source is rejected with
  a pointer to Tracked companies (reuses `detect.Detect`).

## Consequences

- **Purpose is explicit and queryable** via `sources.SourceRole(name)`, decoupled from
  value shape; a future discovery source that happens to use a token, or an ATS reached
  by URL, won't be mis-grouped.
- **Onboarding friction is reduced** — a companion resolver, `detect.ResolveBoard(url)`
  (exposed as `GET /sources/resolve`), turns a pasted direct ATS board URL into a
  `{source, value}` pair so users don't need to know which ATS a company uses. It handles
  direct ATS URLs only; careers pages that redirect to or embed an ATS remain manual entry.
- **No engine change** — periodic re-checking of known companies already worked for any
  ATS board target via `atsPath` on the scrape cron; this ADR only makes the distinction
  legible and safe at the edges.
