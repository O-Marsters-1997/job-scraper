# Capturing and rebasing HTML snapshots

Snapshot fixtures live in `internal/sources/<name>/snapshots/` as paired
`<name>.html` / `<name>.json` files, consumed by `sources.RunSnapshotTests` /
`RunSnapshotTestsURLs`. Never hand-edit the `.json` files — they're generated.

## Capture a page

```
just cli download <source> <name> "<url>"
```

e.g. `just cli download wis page1 "https://workinstartups.com/search?q=engineer"`.

This runs `go run ./cmd/snapshot download <source> <name> <url>` (the `cli`
recipe in `justfile`), fetches the URL with a browser-like User-Agent, and
writes `internal/sources/<source>/snapshots/<name>.html`. Names starting with
`detail_` also get a `<name>.url` sidecar recording the source URL, since detail
parsing (`ParseJobDetail`) needs the URL as well as the HTML.

## Regenerate the JSON fixtures

```
just cli rebase <source>
```

This runs every `*.html` file in the source's `snapshots/` directory through
`ParseURLs` (or `ParseJobDetail`, for `detail_*` files, using the `.url`
sidecar) and writes the matching `.json`. Run it whenever selectors or expected
output change.

`rebase` only knows sources listed in the `parsers` map at the top of
`cmd/snapshot/main.go` — that's why step 4 of the main checklist has you add
the new source there.
