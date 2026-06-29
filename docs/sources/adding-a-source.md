# Adding a New Source

_Last updated: 2026-06-21_

## Step 0: choose source type

The first decision determines almost everything else. Look at the target site:

- **Does it expose a public JSON (or XML) job-board API?** Greenhouse, Lever, Ashby, Workable, Recruitee, and Personio all do. Use an **ATS source** — implement `Source` only. One HTTP call returns all jobs fully populated — no queue, no detail fetch, no HTML parsing.
- **Is it an HTML listing page only?** Use an **HTML source** — implement both `Source` and `DetailFetcher`. The listing page yields URLs and partial cards; a second fetch per URL gets the full detail.

Never use an HTML source when a public API exists. API sources are cheaper, more robust, and skip the queue entirely.

## The source interfaces

```go
// internal/sources/source.go
type Source interface {
    Cfg() Config
    Iterate(ctx context.Context, fn func(ctx context.Context, jobs []dto.Job) (stop bool, err error)) error
}

// DetailFetcher is an optional capability implemented by HTML scrape sources
// that require a separate per-URL fetch to produce a fully-populated dto.Job.
// ATS sources do not implement this interface.
type DetailFetcher interface {
    CanHandle(url string) bool
    GetDetails(ctx context.Context, url string) (dto.Job, error)
}
```

- `Cfg()` — returns name, schedule, `MinScrapeInterval`, `URLPrefix`, and `UseProxy`. Embed `PaginatedBase` and this is provided for free.
- `Iterate` — pages through all jobs. **ATS sources** yield fully-populated `dto.Job`s. **HTML sources** yield `dto.Job{URL: u, Title: ..., Location: ...}` partials with whatever card metadata is available.
- `CanHandle(url)` — (`DetailFetcher` only) returns true if this source owns `url`. `PaginatedBase` implements this as `strings.HasPrefix(url, cfg.URLPrefix)`.
- `GetDetails` — (`DetailFetcher` only) fetch and parse a single job detail page. HTML sources use it as the second-phase fetch.

The orchestrator detects the HTML path via `if _, ok := src.(sources.DetailFetcher); ok`. **ATS sources must not implement `DetailFetcher`.**

### `PaginatedBase`

Embed `sources.PaginatedBase` to get:

- `Cfg()` — returns the `sources.Config` you passed to `sources.NewBase`.
- `CanHandle(url)` — prefix match on `cfg.URLPrefix`.
- `Client()` — the configured `*http.Client` (proxy transport already wired).
- `Get(ctx, url)` — GET with standard headers; returns body bytes or error.
- `IteratePages(ctx, fn, fetchPage, resultsPerPage)` — pagination loop for HTML sources that know their total count. Handles inter-page delays (2–7s random) and context cancellation. `fn` receives `[]string` URLs; wrap them in `dto.Job{URL: u}` before passing up to `Iterate`'s callback.

## Implementing an ATS source

See `internal/sources/greenhouse/greenhouse.go` as the canonical example.

```go
package myats

type Scraper struct {
    sources.PaginatedBase
    cfg Config
}

func New(cfg Config) *Scraper {
    return &Scraper{
        PaginatedBase: sources.NewBase(sources.Config{
            Name:      "myats",
            URLPrefix: "https://boards.myats.io",
            // Schedule and MinScrapeInterval default to 0 */6 * * * and 5h
        }),
        cfg: cfg,
    }
}

// ATS sources implement Source only — do NOT implement DetailFetcher.

func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
    for _, token := range s.cfg.Boards {
        jobs, err := s.fetchBoard(ctx, token)
        if err != nil {
            return fmt.Errorf("myats: board %s: %w", token, err)
        }
        stop, err := fn(ctx, jobs)
        if err != nil || stop {
            return err
        }
    }
    return nil
}
```

Key points:

- Return fully-populated `dto.Job` from `Iterate` (title, location, URL, company slug, source name, description, salary if available).
- Set `Source` field to a stable identifier, e.g. `"greenhouse"`.
- Set `CompanySlug` to the board token — it's used as the dedup-friendly company identifier.
- Do **not** implement `DetailFetcher`. The orchestrator uses interface detection to pick the ATS path; implementing `DetailFetcher` would route jobs through the HTML queue path instead.

## Implementing an HTML source

See `internal/sources/wis/wis.go` as the canonical example. HTML sources require `goquery` for parsing. There is a `goquery-parser` agent skill in this repo — consult it before writing selector logic.

```go
// HTML sources implement both Source and DetailFetcher.

func (s *Scraper) CanHandle(url string) bool { return s.PaginatedBase.CanHandle(url) }

func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
    return s.IteratePages(ctx, func(ctx context.Context, urls []string) (bool, error) {
        jobs := make([]dto.Job, len(urls))
        for i, u := range urls {
            jobs[i] = dto.Job{URL: u} // add Title/Location if available from the card
        }
        return fn(ctx, jobs)
    }, s.fetchPage, resultsPerPage)
}

func (s *Scraper) GetDetails(ctx context.Context, url string) (dto.Job, error) {
    body, err := s.Get(ctx, url)
    if err != nil {
        return dto.Job{}, err
    }
    return ParseJobDetail(bytes.NewReader(body), url)
}
```

**Enrich the listing parse.** The relevance gate runs on `dto.Job.Title` and `dto.Job.Location` from the card. If you return only `dto.Job{URL: u}`, the gate has nothing to score against. Extract whatever the listing page provides.

**Parsing functions.** Keep `ParseURLs` and `ParseJobDetail` as standalone exported functions (not methods) so `RunSnapshotTests` can call them via the `SnapshotSource` interface. Add method forwarders on the scraper:

```go
func (s *Scraper) ParseURLs(r io.Reader) ([]dto.Job, error)      { return ParseURLs(r) }
func (s *Scraper) ParseJobDetail(r io.Reader, url string) (dto.Job, error) { return ParseJobDetail(r, url) }
```

### Proxy

Set `UseProxy: true` in `sources.Config` for sources that hit anti-bot protection (LinkedIn, Indeed). Leave it unset (zero value = false) for ATS APIs and cooperative HTML boards.

When `UseProxy` is true, requests route through BrightData Web Unlocker via `BRIGHTDATA_PROXY_URL`. If the env var is unset the source degrades to direct (no crash). See ADR 0009.

## Snapshot tests

HTML sources require snapshot tests. ATS sources do not (their only parseable artifact is JSON from a live API).

### Capture fixtures

```sh
# Capture a listing page
just cli download <source> list_page1 <listing-url>

# Capture a detail page (also writes a .url sidecar — used by rebase to know the URL)
just cli download <source> detail_job1 <detail-url>

# Generate JSON fixtures from current parser output
just cli rebase <source>
```

Snapshots live in `internal/sources/<source>/snapshots/`. Commit both the `.html` and `.json` files.

### Write the test

```go
// internal/sources/mysource/mysource_test.go
package mysource_test

import (
    "testing"
    "github.com/ollymarsters/job-scraper/internal/sources"
    "github.com/ollymarsters/job-scraper/internal/sources/mysource"
)

func TestSnapshots(t *testing.T) {
    sources.RunSnapshotTests(t, mysource.New())
}
```

`RunSnapshotTests` routes `list_*.html` to `ParseURLs` and `detail_*.html` to `ParseJobDetail`. The URL passed to `ParseJobDetail` comes from the first entry in the corresponding `.json` fixture (so that fixture's `url` field must be set when you run `rebase`).

To update fixtures after a parser change: `just cli rebase <source>`.

## Registration and env-var wiring

Open `cmd/worker/main.go`. Add your source using the same conditional pattern as existing sources:

```go
// ATS source (board-token-based)
if boards := os.Getenv("MYATS_BOARDS"); boards != "" {
    tokens := splitBoards(boards)
    srcs = append(srcs, myats.New(myats.Config{Boards: tokens}))
    slog.Info("myats source registered", slog.Int("boards", len(tokens)))
}

// HTML source (boolean toggle)
if os.Getenv("MYSITE_ENABLED") == "true" {
    srcs = append(srcs, mysite.New())
    slog.Info("mysite source registered")
}
```

`wis` is the only always-on source (no env gate).

Add the new env var to `.env.example` with an empty default and a comment explaining the expected format.

## Checklist

- [ ] ATS source: implements `Source` only (not `DetailFetcher`)
- [ ] HTML source: implements both `Source` and `DetailFetcher` (`CanHandle` + `GetDetails`)
- [ ] `URLPrefix` uniquely matches all URLs this source will produce (HTML sources; used by `CanHandle`)
- [ ] `Iterate` yields partial jobs with at least URL set; title/location set if extractable from listing page
- [ ] `UseProxy` set appropriately in `sources.Config` (true for anti-bot targets)
- [ ] Snapshot tests added and passing (HTML sources only)
- [ ] Registered in `cmd/worker/main.go` behind an env var
- [ ] Env var added to `.env.example`
