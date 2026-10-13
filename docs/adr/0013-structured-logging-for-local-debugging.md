# ADR 0013 — Structured logging: one injected logger, errors logged once at the top

## Context

The main reader of our logs is a developer running `just run` and `just run-api` and following a
scrape through both processes. They need one ID that joins a request, or a Scrape Run, across the api
and the worker, shared attribute keys, and each failure logged once with its origin.

Prod ships the same stdout to Grafana Cloud through Alloy (ADR 0010). Loki bills by volume, and
new labels multiply streams, so nothing here may add labels or much volume. The go-idiomatic skill
(`references/errors.md`, logging) is the authority for the rules below.

## Decision

`internal/logger` is the only place that knows how a log line is built. Its interface:

```go
func New(w io.Writer, format, level string) (*slog.Logger, error)
func With(ctx context.Context, attrs ...slog.Attr) context.Context
func Middleware(next http.Handler) http.Handler
func Transport(base http.RoundTripper) http.RoundTripper
const KeyErr, KeyRunID, KeyTaskID, KeyRequestID, KeySource, … // every attribute key
```

- **`main` builds one logger and injects it at the top of the chain.** `cmd/api` and `cmd/worker` read
  `LOG_FORMAT` (`json` by default, or `text`, which uses the stdlib `TextHandler`) and `LOG_LEVEL` (`info`
  by default) into their `Config`, call `New`, and pass the result to the top-of-chain components:
  `writeError`, worker processors, queue consumers and schedulers. Unknown values return an error, so a
  typo fails at startup. The `just run*` recipes set `LOG_FORMAT=text LOG_LEVEL=debug`. The CLIs (`queue`,
  `snapshot`, `admin` output) keep `fmt`.
- **Lower code returns errors and does not log.** Stores, services and helpers wrap and return. There is
  no `slog.Default`, no `slog.SetDefault` and no package-level `slog.*` call; `forbidigo` enforces it.
- **The origin of an error is captured once, at the seam where it enters the process** (`errsite.Mark`:
  store, outbound HTTP, queue publish). The top-of-chain component logs the error once, with the origin
  `file:line` as `err_site`.
- **Wraps are short: `<verb> <noun> <id>: %w`.** They carry no `pkg.Func:` prefix, because `err_site`
  locates the origin.
- **`With`** returns a ctx carrying attributes. The handler `New` builds appends them to every record
  logged through a `*Context` call (`log.InfoContext(ctx, …)`), and a key set at the call site wins over
  the same key on ctx.
- **Where attributes are set:** the broker's consume loop calls `With` with `run_id`, `task_id`, `source`
  and `kind` before running the handler. `Middleware`, mounted after chi's `middleware.RequestID`, adds
  `request_id`, a fresh `trace_id`, and reads `run_id` from an `X-Run-ID` header. `Transport` writes ctx's
  `run_id` onto that header. Only the worker's API client uses `Transport`: a source fetch never sends it
  to a third party. `trace_id` is a UUID minted per request, not tied to chi's process-local counter, so it
  stays unique across a restart or redeploy for grepping one request's lines.
- **Shared keys are consts** in `internal/logger`: correlation IDs, and every key `ops/grafana` queries
  (`event`, `route`, `source`, `kind`, `user_id`, `outcome`, `cost_usd`, `duration_ms`, `wait_ms`).
  `route` means the chi route pattern everywhere. A key only one call site uses keeps its own string
  literal, because `sloglint` doesn't enforce going through the registry; a second call site wanting the
  same field is the cue to add a const.
- **`sloglint` enforces the mechanical part** in `.golangci.yml`: `attr-only`, `key-naming-case: snake`,
  `context: scope`, `static-msg`, and `forbidden-keys` for `time`, `level` and `msg`.
- **Extra lines go at boundaries, at DEBUG:** outbound source fetches (url, status, duration), queue
  consume/ack and ingest counts. Services and stores add none.

Rejected:

- A logger carried on ctx (`logger.From(ctx).Info`). Every call site changes shape, and a missing logger
  falls back silently.
- A global default logger. Any call site could log, so one failure would be logged at every layer it
  passes through.
- W3C `traceparent` over AMQP and HTTP for correlation. Grep on `run_id` covers the need.
- `tint` for colour. It's a dependency, and the stdlib text handler reads well enough.
- Pretty-printing by piping JSON through `jq` in the recipes. The pipe hides the exit code.

## Consequences

- Attributes on ctx stay in the JSON body. Loki labels remain `service`, `level` and `env` (ADR 0010),
  so ingest cost only moves with volume. With prod at `info`, DEBUG lines are free there.
- `grep run_id=<id>` in text output, or `jq 'select(.run_id=="<id>")'` in JSON, follows one Scrape Run
  from consume, through fetch, to the api's ingest handler.
- Each failure appears once, with `err_site` pointing at where it began.
- A log call without ctx drops the ctx attributes. `sloglint`'s `context: scope` catches this wherever a
  ctx is in scope.
- A new cross-process hop needs `Transport` on the client and `Middleware` on the server, or the chain
  breaks at that hop.
- The `instrument-logging` skill describes how to follow this ADR.
