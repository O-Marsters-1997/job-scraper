---
name: add-source
description: Add a new job source, either an ATS board API (Greenhouse, Lever, Ashby…) or an HTML listing site. Use for "add a source", "scrape X", "new job board", "support <site>".
paths: ["internal/worker/sources/**", "cmd/worker/**", "cmd/snapshot/**"]
---

# Add a Job Source

Two branches, depending on what the target site offers.

## 1. Pick a branch

**ATS source** — a public JSON/XML board API (Greenhouse, Lever, Ashby, Workable,
Recruitee, Personio). Copy `internal/worker/sources/greenhouse/greenhouse.go` (~65 lines):
it builds a `sources.BoardSpec` and calls `sources.NewBoardSource`, implementing
`Source` only.

**HTML source** — a listing page with no API. Copy `internal/worker/sources/wis/wis.go`:
it embeds `sources.PaginatedBase` and implements `Iterate` plus `DetailFetcher`
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

Add the source to `BuildSources` in `internal/worker/sources/builder/build.go`, following
the pattern for the existing board/URL/filter sources there. This file has no
compile-time safety net — skip this step and the source is registered but
silently never runs.

ATS sources: save a real API response to `snapshots/<name>.json` and test `parse` with
`sourcetest.RunGolden(t, "<name>.json", "<token>", parse)`. Create or refresh the
`<name>.golden.json` with `go test ./internal/worker/sources/<source> -update` and review the diff.

## 4. HTML sources only

- Add the source to the `detailers` map in `cmd/worker/main.go` — detail fetching
  isn't automatic.
- Add it to the `parsers` map in `cmd/snapshot/main.go` so the snapshot CLI knows
  how to parse its fixtures.
- Add snapshot tests under `internal/worker/sources/<name>/snapshots/`. See
  `references/snapshots.md` for the exact capture/rebase commands.

## 5. Verify

```
go test ./internal/worker/sources/...
```

`TestBuildSources_EveryRegisteredSourceInstantiates`
(`internal/worker/sources/builder/build_test.go`) is what catches a missing step 3 —
it fails if a registered source never gets built by `BuildSources`.
