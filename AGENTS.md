# AGENTS.md

## Approach

- Think before acting; read existing files before writing code. Prefer editing over rewriting.
- Test your code before declaring done. Comments follow `~/.claude/rules/comments.md`.
- No sycophantic openers or closing fluff. Be concise in output, thorough in reasoning.
- Before a structural/architectural decision (new package boundary, naming ambiguity), check
  `CONTEXT.md` (domain glossary) and `docs/adr/` (past decisions) — not needed for routine work.
- Use **bun** exclusively for JS/TS package management (`frontend/`, `emails/`). Never npm/pnpm/yarn.

## Layout

Modular monolith split by context ([ADR 0011](docs/adr/0011-modular-monolith-by-context.md), which has the
full layout, port and enforcement rules): `jobsearch`, `scoring`, `applications`, `cvtemplates`, `identity`.

- `cmd/<binary>`: entrypoints only. `cmd/api/main.go` is the only composition root.
- `internal/services/<ctx>/`: one package per context with `module.go` (`New(deps)` and the facade),
  `routes.go` and `service.go`. Add a feature file, not a feature package. `store/` (with `queries/` and
  generated `sqlc/`) is imported only by its own context (`<ctx>-store` depguard rule).
- `internal/api/`: HTTP shell only, used by `cmd/api`. `internal/worker/`: used by `cmd/worker`; imports
  context roots, never `internal/api` (ADR 0009).
- Everything else under `internal/` is the shared kernel and imports no context, `internal/api` or
  `internal/worker`. `internal/data/db` connects and migrates; `internal/data` holds only genuinely
  reusable pieces (e.g. `ErrNotFound`).

Rules:

- Only the owning context writes its tables; any store may SELECT-join others. A cross-context write in one
  transaction goes through that context's tx-scoped port (`scoring.JobsChanged(ctx, tx, jobIDs)`).
- `dto` holds only HTTP shapes and types that cross a facade.
- Place new code by the table map in ADR 0011. If a feature doesn't fit a context, stop and ask.

## Adding a new source

Use the `add-source` skill. It covers the ATS vs. HTML branch, registry/builder wiring, and
snapshots.

## Anatomy of a handler

See [ADR 0008](docs/adr/0008-handlers-over-feature-services.md):

- Every handler is built from `Handle(decode, call, respond)` (`internal/handlers/generic.go`),
  either directly or through one of the CRUD-shaped generics (`GetAll`, `GetByID`, `Query`, `Create`,
  `Update`, `Delete`). There is no third way. Routes are registered in the owning module's `Routes`.
  Handlers hold no business logic and don't log; only `writeError` logs, and only for an error with no
  `apperr` kind.
- Domain validation and orchestration live in `internal/services/<ctx>` (or a split-out `internal/services/<feature>`). A service's
  dependencies are required constructor args: a store interface declared in the service's own package,
  plus small local interfaces for anything else (queue publisher, verifier, another module's facade).
  No `With*` setters.
- Request bodies decode into `dto` input types; path IDs fill `path:"…"`-tagged dto fields after
  decoding, and the user ID is a service arg, so a body can never set either. Services return
  `apperr` errors and wire-ready `dto` values.
- A route with no logic binds a CRUD generic directly to a store method value. Add a service method
  only when there's a rule or orchestration to hold.
- Routes that set cookies, redirect, stream, or use service-token auth (login/signup/logout/me, OAuth
  start/callback, CV export, ingest) call `Handle` directly with their own decode/call/respond.

Use the `new-handler` skill for the end-to-end steps, backend and frontend.

## sqlc

- Schema lives in `internal/data/db/sqlc/schema.sql`. sqlc never reads `scripts/migrations/`, so mirror
  every migration there by hand.
- Each context has one `sql:` block in `sqlc.yaml`: queries in `internal/services/<ctx>/store/queries/`,
  generated into `internal/services/<ctx>/store/sqlc/`.
- `just generate` (`sqlc generate`) regenerates all generated trees; never hand-edit them.
  CI runs `sqlc generate && git diff --exit-code`, so commit generated code with the schema change.

Use the `schema-change` skill for the full migration → sqlc → store procedure.

## Tests

Load the `testing-policy` skill before writing or changing any test: it maps each kind of code to its
test, doubles and assertions ([ADR 0012](docs/adr/0012-test-seams-and-double-packages.md)).

## Commands

See `justfile` (`just --list`) for build, lint, test, migration, and snapshot commands.
DB-backed tests need Docker (`just up`).

## Traces

Run `just tracing-up`, set `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`, then `just run-api` /
`just run` (ADR 0014). Query Tempo at `http://localhost:3200`:

```
curl -G localhost:3200/api/search --data-urlencode 'q={resource.service.name="worker" && status=error}'
curl -G localhost:3200/api/search --data-urlencode 'q={span.http.response.status_code>=500}' --data-urlencode limit=20
curl -G localhost:3200/api/search --data-urlencode 'q={duration>2s}'
curl localhost:3200/api/traces/<trace_id>   # trace_id from a log line
```

## Gotchas

- Migrations resolve `scripts/migrations` relative to CWD. Don't run binaries or DB tests from
  elsewhere; do run from the repo root (`pgtest` and `RunMigrations` need it).
- Don't edit an applied migration in `scripts/migrations/`; do add a new one with `just migrate-create`.
- Use the domain terms in `CONTEXT.md` (Source Target, Board, Candidate, Scrape Run); read the
  relevant `docs/adr/` before changing an area.

## Boundaries — never hand-edit

- `internal/services/*/store/sqlc/**` → `sqlc generate`
- `internal/api/notify/templates/*.tmpl` (moves to `internal/services/notify/` with scoring) → edit
  `emails/templates/*.tsx`, run `just build-emails`
- `frontend/src/routeTree.gen.ts`     → regenerated by the TanStack router Vite plugin
- `frontend/src/components/ui/**`     → vendored (Zaidan/Kobalte); re-run `bun run add-component <name>`
- `internal/worker/sources/*/snapshots/*.json` → `just cli rebase <source>`
- `.env` (secrets); document new vars in `.env.example`
- `bun.lock` files → only via bun (never npm/pnpm/yarn)

## Agent skills

- Issue tracker: `docs/agents/issue-tracker.md`
- Triage labels: `docs/agents/triage-labels.md`
- Domain docs: `docs/agents/domain.md`

## UI / design

Read `DESIGN.md` (repo root) before any frontend UI change. See also `frontend/AGENTS.md`.
