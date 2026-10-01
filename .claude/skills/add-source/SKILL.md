---
name: add-source
description: Add a new job source, either an ATS board API (Greenhouse, Lever, Ashby…) or an HTML listing site. Use for "add a source", "scrape X", "new job board", "support <site>".
---

# Add a Job Source

Two branches, depending on what the target site offers.

## 1. Pick a branch

**ATS source** — a public JSON/XML board API (Greenhouse, Lever, Ashby, Workable,
Recruitee, Personio). Copy `internal/worker/sources/greenhouse/greenhouse.go` (~65 lines):
it builds a `sources.BoardSpec` and calls `sources.NewBoardSource`, implementing
`Source` only.

**HTML source** — a listing page with no API. Copy `internal/worker/sources/wis/wis.go`:
it embeds `sources.PaginatedBase` and implements `FetchPage` plus `DetailFetcher`
(`CanHandle`/`GetDetails`). Load the `goquery-parsing` skill before writing any
selectors — see `references/html-parsing.md` for when.

Do not copy `internal/worker/sources/indeed` for either branch — it predates both
patterns and is a legacy outlier, not something to imitate.

## 2. Register it

Add an entry to the `entries` slice in `internal/sourcespec/sourcespec.go` (metadata the API
and worker both read; it must not import any adapter):

- `name` — must equal the name baked into the source's own `Config`/`BoardSpec`.
- `kind` — `kindBoard` (value is a board token), `kindURL` (value is a full URL),
  or `kindFilter` (value is a keyword plus structured `filters`).
- `role` — `RoleATS` or `RoleDiscovery`.

## 3. Wire instantiation

Add one entry to `registry` in `internal/worker/sources/builder/build.go`: `build` makes the
listing Source; `detail` is its `DetailFetcher` if it has one; `cardComplete: true` means the
listing card is already the full job and no detail fetch is needed. `cmd/worker` and the
Processor read the registry, so nothing else lists sources. Skip this step and the source is
registered but silently never runs; `TestBuildSource_EveryRegisteredSourceInstantiates` catches it.

ATS sources: save a real API response to `snapshots/<name>.json` and test `parse` with
`sourcetest.RunGolden(t, "<name>.json", "<token>", parse)`. Create or refresh the
`<name>.golden.json` with `go test ./internal/worker/sources/<source> -update` and review the diff.

## 4. HTML sources only

- Set `detail` on its registry entry (step 3) so the worker can fetch details.
- Add it to the `parsers` map in `cmd/snapshot/main.go` so the snapshot CLI knows
  how to parse its fixtures.
- Add snapshot tests under `internal/worker/sources/<name>/snapshots/`. See
  `references/snapshots.md` for the exact capture/rebase commands.

## 5. Verify

```
go test ./internal/worker/sources/...
```
