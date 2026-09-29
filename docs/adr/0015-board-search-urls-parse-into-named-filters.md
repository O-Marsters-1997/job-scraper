# ADR 0015 — Board search URLs parse into named filters

## Context

Users paste the search page they already have open on LinkedIn, Indeed or Work in Startups. The
frontend parsed these itself (`boardUrl.ts`) with its own copy of each board's parameter names, so
the vocabulary lived in two places and the backend still stored whatever string it was given: two
Indeed URLs differing only in `vjk=` or `from=` became two Source Targets.

## Decision

`internal/detect` owns both directions:

- `ParseSearchURL(raw) (SearchURL, bool)` maps a pasted URL to `{source, value, filters, dropped}`.
  URL params map to the `sourcespec` filter names (`f_TPR` to `recency`, `geoId` to `geo_id`, `w` to
  `region`). Enumerated values must be declared options and ID fields must be numeric; anything else
  is reported in `dropped`, as are tracking params (`currentJobId`, `origin`, `referralSearchId`,
  `utm_*`). Paging params (`start`, `p`) are discarded silently. LinkedIn `/jobs/search-results/` is
  treated as `/jobs/search/`.
- `BuildSearchURL(source, value, filters) string` returns the canonical board page, used for the
  "Open on board" link. `BuildSearchURL(ParseSearchURL(u))` is stable.
- Indeed is a URL-kind source with no declared filters, so its `value` is the normalised URL: only
  `q`, `l`, `fromage`, `radius`, `jt` and `sort` survive, in sorted order. `POST /source-targets`
  runs Indeed values through the same code, so tracking-param variants hit the unique key and return
  409. Existing rows are re-normalised when read, not migrated.

`GET /sources/resolve` returns `dto.ResolvedURL {kind: "search"|"ats", source, value, filters,
dropped, url}` for both search pages and ATS boards. RemoteOK and Remotive URLs get a "not supported
yet" 422, distinct from the 422 for an unrecognised URL. `GET /source-targets` fills
`dto.SourceTarget.URL` through `sourcetargets.Service.List`.

## Consequences

- A new filter is declared once in `sourcespec`, then mapped to its URL param in `detect`.
- Dropped params are visible to the user rather than silently ignored.
- RemoteOK and Remotive searches can only be built from fields.
