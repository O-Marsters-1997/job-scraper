# ADR 0011 — Split `DetailFetcher` capability out of the `Source` interface

**Status:** Accepted

## Context

The original `Source` interface included `NeedsDetail() bool`, `CanHandle(url string) bool`, and `GetDetails(ctx, url)` on every source. This forced ATS sources to provide stub implementations of methods they can never legally be called on (`GetDetails` returning an error, `NeedsDetail` returning `false`). It also encoded the HTML/ATS branch as a runtime boolean rather than a type-system distinction.

The interface was also larger than necessary for callers that only needed to schedule scrapes (`Cfg`, `Iterate`) and had no interest in URL dispatch or detail fetching.

## Decision

Split the monolithic `Source` interface into two:

```go
// Every source implements this.
type Source interface {
    Cfg() Config
    Iterate(ctx context.Context, fn func(ctx context.Context, jobs []dto.Job) (stop bool, err error)) error
}

// Optional capability; HTML sources implement it, ATS sources do not.
type DetailFetcher interface {
    CanHandle(url string) bool
    GetDetails(ctx context.Context, url string) (dto.Job, error)
}
```

The orchestrator determines the code path via type assertion: `if _, ok := src.(sources.DetailFetcher)`. If true, the source is treated as HTML and its URLs are enqueued for the worker. If false, the source is ATS and jobs are exported directly.

`Dispatch(ctx, detailers, url)` in `source.go` receives `[]DetailFetcher` (assembled from the source list in `cmd/worker`) and routes a URL to the first fetcher whose `CanHandle` returns true.

## Consequences

- **ATS sources are simpler** — no stub `GetDetails`, no `NeedsDetail` boolean, no `CanHandle` on a type that never handles URLs.
- **Capability is declared by implementation, not by a flag** — the Go type system enforces the contract; a bug that makes an ATS source accidentally implement `DetailFetcher` is caught at the orchestrator branch, not silently swallowed.
- **`CanHandle` moves to `DetailFetcher`** — callers that only route HTML detail URLs (`Dispatch`) only see fetchers that can handle them. `PaginatedBase.CanHandle` (prefix match on `URLPrefix`) is still the default implementation, forwarded by HTML sources via a one-liner.
- **Decomposed `Orchestrator.run()`** — the internal ATS and HTML paths are now separate `atsPath` and `htmlPath` structs with `onPage` callbacks, making each path independently readable and testable.
