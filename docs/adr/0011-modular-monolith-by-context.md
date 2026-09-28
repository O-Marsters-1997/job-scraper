# ADR 0011 — A modular monolith split by context, each owning its tables

One `*jobsdb.DB` implemented every `providers` interface, so every service could reach every table and constructors took `db` several times over (`suitability.New(db, db, db, …, db)`). The goal is locality and data ownership inside one process and one database. Pulling a context out into its own service is not planned.

- **Contexts.**
  - `jobsearch` decides what to fetch: jobs, job URLs, companies, boards, poll state, candidates and Relevance, harvest runs, source targets with their run state, and tracked companies.
  - `scoring` decides how well a job fits: search config, options, answers, scores, `effect_outbox`, Jev and notify. The answer-effect loop becomes `scoring.Run(ctx)`.
  - `applications`: applications and statuses.
  - `cvtemplates`: tracked docs and tabs.
  - `identity`: users, sessions, profile, AI credentials, and the Google Link. The Google Link covers OAuth, tokens and the Docs client, which identity exposes to cvtemplates.
- **Layout.** Every context lives under `internal/services/`.
  - `internal/services/<ctx>/module.go`: `New(deps) *Module`, where `deps` are required constructor args.
  - `Module` has a narrow facade: exported methods for what other contexts, the worker or `cmd/admin` need, taking and returning `dto` types.
  - `m.Routes(r chi.Router)`, in `internal/services/<ctx>/routes.go`, binds the context's routes with `handlers.Handle` (ADR 0008).
  - The context's main feature service is `internal/services/<ctx>/service.go` (`NewService`). A context is one package: one `Service`, one `Store` interface, and each feature a file on that `Service`. A feature gets its own package only when the store returns its types, so folding it into the root would be an import cycle: `sourcetargets` owns `Candidate`, which `jobsearch/store` returns. Such a package declares its own store interface and never imports the store. `Build(Deps)` takes the store and one field per other boundary: external clients and other contexts' ports.
  - `internal/services/<ctx>/store/` holds the store (`store.go`), its sqlc-to-`dto` converters (`transform.go`, named `to<Name>DTO`), its queries and its generated sqlc.
  - `cmd/api/main.go` is the only composition root. `internal/api` shrinks to the HTTP shell: middleware, CORS, and mounting each module's routes.
  - `handlers.Handle` and the CRUD generics move to `internal/handlers`, so every context can import them.
- **Shared kernel.** Packages stay flat under `internal/`: `dto`, `apperr`, `queue`, `telemetry`, `handlers`, `pgtest`, plus existing helpers such as `sourcespec` and `slug`. `dto` keeps only HTTP shapes and types that cross a facade. DB input structs move into their context (amends ADR 0001). The kernel never imports a context: `telemetry.StateCollector` takes an interface that scoring satisfies.
- **Writes are owned, reads may join.**
  - Only the owning context writes its tables. Any store may SELECT-join another context's tables, which keeps score-sorted job pagination in one query.
  - A write that must change another context's rows in the same transaction goes through a transaction-scoped port that context exports. Scoring exports three: `JobsChanged(ctx, tx, jobIDs)` (a job's content changed), `JobsClosed(ctx, tx, jobIDs)` (a job closed) and `CompanyTracked(ctx, tx, userID, companyID)` (a user started tracking a company). `applications.SeedDefaults(ctx, tx, userID)` serves signup.
- **Enforcement.**
  - `sqlc.yaml` has one `sql:` block per context. All of them read the shared `internal/data/sqlc/schema.sql`.
  - Queries live in `internal/services/<ctx>/store/queries/` and generate into `internal/services/<ctx>/store/sqlc/`, so a context cannot call another context's write queries.
  - `depguard` keeps each store private. A `<ctx>-store` rule per context denies `internal/services/<ctx>/store` outside `internal/services/<ctx>/`, and the `services` rule denies the legacy `providers` and `data/db` to every context. This is enforced by lint, not by the compiler.
  - The worker and `cmd/admin` import context roots only and never call `Routes`. ADR 0009's rule stands, and `depguard` keeps enforcing it.
- **Tests.**
  - `providers` and `providers.Mock*` are removed. Each service declares its own store interface and is tested with small hand fakes.
  - Stores are tested against Postgres through the shared `internal/pgtest` helper (amends ADR 0008's tests bullet).
- **Errors.**
  - A store returns sentinels declared in its own package, kinded with `apperr` when they map to an HTTP status, and maps `pgx.ErrNoRows` to its own `ErrNotFound`.
  - The context root (`internal/services/<ctx>`) re-exports only the sentinels that other contexts or the worker must match. The root can't declare them itself, because it imports the store.
  - Amendment (2026-09-28): `ErrNotFound` carried no domain-specific detail in any store, so four
    independently-built copies were silently not `errors.Is`-equal to each other. It's now one
    `data.ErrNotFound` (`internal/data/errors.go`), returned directly — no per-store declaration
    or root re-export. A sentinel stays local only when it carries real domain meaning
    (`ErrApplicationExists`, `ErrUsernameTaken`, ...).
- **Migration order.** `applications` is the pilot, followed by `identity`, `cvtemplates` and `scoring`. `jobsearch` goes last, taking whatever remains of `internal/data/db`. Until a context moves, `*jobsdb.DB` keeps serving it.

Rejected alternatives:

- A central router binding facade methods: facades would grow to one method per endpoint.
- A separate `<ctx>/httpapi` package: it can't reach the service instances `New` builds without a second wiring graph.
- Outbox events for cross-context invalidation: Suitability staleness should stay atomic with job edits.
- Composing reads in Go: breaks score-sorted pagination.
- Per-context Postgres schemas and roles: more than a locality goal needs.
- One shared `pgsqlc`: ownership would be enforced only by review.
- Per-context mock packages: a wide store interface would come back.
- A sibling package per feature, joined by ports: each had one consumer, its own context, and the ports (`Reconsiderer`, `CredentialLister`, ...) existed only to connect siblings.
- `internal/<ctx>/internal/<feature>` and `internal/<ctx>/internal/store`, so Go's `internal/` rule enforces store privacy: the applications pilot found the doubled `internal` hard to read, and depguard covers the same boundary.

Trade-off: transaction-scoped ports put `pgx.Tx` in interfaces that cross contexts, and read-joins mean a context can't change its tables freely without checking who reads them. sqlc also generates duplicate model structs in each package, which get mapped to `dto` anyway.

Status: complete (2026-09-28, PR #305) — `internal/data/db`, `internal/data/providers` and the legacy sqlc block removed.
