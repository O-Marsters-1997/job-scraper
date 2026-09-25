# AGENTS.md

## Approach

- Think before acting; read existing files before writing code. Prefer editing over rewriting.
- Test your code before declaring done. Only add comments if the code isn't self-descriptive.
- No sycophantic openers or closing fluff. Be concise in output, thorough in reasoning.
- Before a structural/architectural decision (new package boundary, naming ambiguity), check
  `CONTEXT.md` (domain glossary) and `docs/adr/` (past decisions) — not needed for routine work.
- Use **bun** exclusively for JS/TS package management (`frontend/`, `emails/`). Never npm/pnpm/yarn.

## Adding a new source

1. **Pick the source type.** Public JSON/XML board API (Greenhouse, Lever, Ashby, Workable,
   Recruitee, Personio) → **ATS source**: build a `sources.BoardSpec` and call
   `sources.NewBoardSource` — copy `internal/sources/greenhouse/greenhouse.go` (65 lines),
   implements `Source` only. HTML listing page only → **HTML source**: embed
   `sources.PaginatedBase`, implement `Iterate` + `DetailFetcher` (`CanHandle`/`GetDetails`) —
   copy `internal/sources/wis/wis.go`. Don't copy `internal/sources/indeed`; it's a legacy outlier.
2. **Register it** in `internal/sources/registry/registry.go`'s `entries` slice: `name` (must
   equal the name baked into the source's own `Config`/`BoardSpec`), `kind` (`kindBoard` = board
   token, `kindURL` = full URL, `kindFilter` = keyword + structured `filters`), and `role`
   (`RoleATS` / `RoleDiscovery`).
3. **Wire instantiation** in `internal/sources/builder/build.go`'s `BuildSources` — this has no
   compile-time safety net; skip it and the source silently never runs.
   `TestBuildSources_EveryRegisteredSourceInstantiates` catches a missing wire-up: run
   `go test ./internal/sources/builder/`.
4. **HTML sources only:** add the source to the `detailers` map in `cmd/worker/main.go` (detail
   fetching isn't automatic), and add snapshot tests under `internal/sources/<name>/snapshots/`
   (capture with `just cli download`, regenerate with `just cli rebase`).

## Anatomy of a handler

Pending rollout of [ADR 0020](docs/adr/0020-handlers-as-http-adapter-over-feature-services.md)
(accepted, in progress) — this is the target shape; don't copy the pre-ADR inline style from
files not yet migrated:

- `internal/handlers` is a thin HTTP adapter only: generic wrappers `User`/`ID`/`Body`/`BodyID`
  handle the session, decoding, `apperr` kind → status mapping, and JSON encoding. Handlers hold
  no business logic and don't log — only the adapter logs.
- Domain validation and orchestration live in per-feature packages (`internal/companies`,
  `internal/sourcetargets`, ...). A service's dependencies are required constructor args —
  `providers.X` for persistence, small interfaces declared in the service's own package for
  anything else (queue publisher, verifier, scorer). No `With*` setters.
- Request bodies decode into `dto` input types; the caller's user ID and path IDs are passed as
  service args (`ctx, userID, id…, in`), never as dto fields. Services return `apperr` errors and
  wire-ready `dto` values.
- A route with no logic binds the adapter directly to a provider method value — add a service
  method only when there's a rule or orchestration to hold.
- Routes that set cookies, redirect, stream, or use service-token auth stay as plain
  `http.HandlerFunc`s using the shared `Caller`/`DecodeJSON`/`WriteJSON`/`WriteError` helpers.

## sqlc

- Schema lives in `internal/data/sqlc/schema.sql`, queries in `internal/data/sqlc/queries/*.sql`
  — sqlc reads neither from `scripts/migrations/`; mirror every migration there by hand.
- `just generate` (`sqlc generate`) regenerates `internal/data/db/pgsqlc/**` — never hand-edit it.
  CI runs `sqlc generate && git diff --exit-code`, so commit generated code with the schema change.
- Wrap generated calls in `internal/data/db/*.go`: map `pgx.ErrNoRows` → `providers.ErrNotFound`,
  wrap other errors as `fmt.Errorf("db.Method: %w", err)`. Copy `internal/data/db/profile.go`.
  Inputs are DTOs (ADR-0001).

## Tests

- DB tests share `testDB` from `internal/data/db/db_test.go` (a real Postgres testcontainer) —
  don't start a second container.
- Once ADR 0020 lands, services are tested with `providers.Mock*`
  (`internal/data/providers/mock_*.go`) plus small fake ports in the service package, without
  HTTP — prefer `providers.Mock*` over a local `fake`/`mock` struct.

## Commands

See `justfile` (`just --list`) for build, lint, test, migration, and snapshot commands.
DB-backed tests need Docker (`just up`).

## Gotchas

- Migrations resolve `scripts/migrations` relative to CWD. Don't run binaries or DB tests from
  elsewhere; do run from the repo root (`db_test.go` chdirs there for this reason).
- Don't edit an applied migration in `scripts/migrations/`; do add a new one with `just migrate-create`.
- Use the domain terms in `CONTEXT.md` (Source Target, Board, Candidate, Scrape Run); read the
  relevant `docs/adr/` before changing an area.

## Boundaries — never hand-edit

- `internal/data/db/pgsqlc/**`        → `sqlc generate`
- `internal/notify/templates/*.tmpl`  → edit `emails/templates/*.tsx`, run `just build-emails`
- `frontend/src/routeTree.gen.ts`     → regenerated by the TanStack router Vite plugin
- `frontend/src/components/ui/**`     → vendored (Zaidan/Kobalte); re-run `bun run add-component <name>`
- `internal/sources/*/snapshots/*.json` → `just cli rebase <source>`
- `.env` (secrets); document new vars in `.env.example`
- `bun.lock` files → only via bun (never npm/pnpm/yarn)

## Agent skills

- Issue tracker: `docs/agents/issue-tracker.md`
- Triage labels: `docs/agents/triage-labels.md`
- Domain docs: `docs/agents/domain.md`

## UI / design

Read `DESIGN.md` (repo root) before any frontend UI change. See also `frontend/AGENTS.md`.
