# ADR 0013 — Structured logging for local debugging

## Context

The main reader of our logs is a developer running `just run` and `just run-api` and following a
scrape through both processes. Today that's hard:

- `internal/logger` always writes JSON at DEBUG.
- No ID joins a request, or a Scrape Run, across the api and the worker. `run_id` is logged in
  the broker but not sent on the worker's ingest call.
- Every call site repeats its own `task_id`/`run_id`/`source` attributes.
- Keys drift: `company` vs `company_id`, `board` vs `board_id`, `x-delivery-count`, and
  `route` meaning the chi pattern in `AccessLog` but the raw path in `writeError`.

Prod ships the same stdout to Grafana Cloud through Alloy (ADR 0010). Loki bills by volume, and
new labels multiply streams, so nothing here may add labels or much volume.

## Decision

`internal/logger` is the only place that knows how a log line is built. Its interface:

```go
func New(w io.Writer, format, level string) (*slog.Logger, error)
func With(ctx context.Context, attrs ...slog.Attr) context.Context
func Middleware(next http.Handler) http.Handler
func Transport(base http.RoundTripper) http.RoundTripper
const KeyErr, KeyRunID, KeyTaskID, KeyRequestID, KeySource, … // every attribute key
```

- **`New`** takes `LOG_FORMAT` (`json` by default, or `text`, which uses the stdlib
  `TextHandler`) and `LOG_LEVEL` (`info` by default). Unknown values return an error, so a typo
  fails at startup. `cmd/api` and `cmd/worker` read the env and call `slog.SetDefault`. The
  `just run*` recipes set `LOG_FORMAT=text LOG_LEVEL=debug`. The CLIs (`queue`, `snapshot`,
  `admin` output) keep `fmt`.
- **`With`** returns a ctx carrying attributes. The handler `New` builds appends them to every
  record logged through a `*Context` call (`slog.InfoContext(ctx, …)`), and a key set at the
  call site wins over the same key on ctx. Code keeps using the global default. Nothing
  injects a logger.
- **Where attributes are set:** the broker's consume loop calls `With` with `run_id`,
  `task_id`, `source` and `kind` before running the handler. `Middleware`, mounted after chi's
  `middleware.RequestID`, adds `request_id`, a fresh `trace_id`, and reads `run_id` from an
  `X-Run-ID` header. `Transport` writes ctx's `run_id` onto that header. Only the worker's API
  client uses `Transport`: a source fetch never sends it to a third party. `trace_id` is a UUID
  minted per request, not tied to chi's process-local counter, so it stays unique across a
  restart or redeploy for grepping one request's lines.
- **Shared keys are consts** in `internal/logger`: correlation IDs, and every key `ops/grafana`
  queries (`event`, `route`, `source`, `kind`, `user_id`, `outcome`, `cost_usd`, `duration_ms`,
  `wait_ms`). `route` means the chi route pattern everywhere. A key only one call site uses
  keeps its own string literal — `sloglint` doesn't enforce going through the registry, so this
  is a convention, not a guarantee; a second call site wanting the same field is the cue to add
  a const.
- **`sloglint` enforces the mechanical part** in `.golangci.yml`: `attr-only`, `key-naming-case:
  snake`, `context: scope`, `static-msg`, and `forbidden-keys` for `time`, `level` and `msg`.
- **New lines go at boundaries, at DEBUG:** outbound source fetches (url, status, duration),
  queue consume/ack and ingest counts. Services and stores add none.

Rejected:

- A logger carried on ctx (`logger.From(ctx).Info`). Every call site changes shape, and a
  missing logger falls back silently.
- Injecting `*slog.Logger` through constructors. Too much churn for no second adapter.
- W3C `traceparent` over AMQP and HTTP. ADR 0010 deferred traces, and grep on `run_id` covers
  the need.
- `tint` for colour. It's a dependency, and the stdlib text handler reads well enough.
- Pretty-printing by piping JSON through `jq` in the recipes. The pipe hides the exit code.

## Consequences

- Attributes on ctx stay in the JSON body. Loki labels remain `service`, `level` and `env`
  (ADR 0010), so ingest cost only moves with volume. With prod at `info`, DEBUG lines are free
  there.
- `grep run_id=<id>` in text output, or `jq 'select(.run_id=="<id>")'` in JSON, follows one Scrape
  Run from consume, through fetch, to the api's ingest handler.
- A `slog.Info` call without ctx silently drops the ctx attributes. `sloglint`'s
  `context: scope` catches this wherever a ctx is in scope.
- A new cross-process hop needs `Transport` on the client and `Middleware` on the server, or
  the chain breaks at that hop.
- The `instrument-logging` skill describes how to follow this ADR. Its Valkey, cron and
  store-DEBUG guidance is removed.
