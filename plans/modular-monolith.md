# Plan: Modular monolith by context

> Source: [ADR 0011](../docs/adr/0011-modular-monolith-by-context.md) and `AGENTS.md` § Layout. This is a refactor: routes, response shapes and runtime behaviour must not change.

## Technical design decisions

- **Routes.** Every HTTP path and response shape stays the same, and the frontend must not notice. Routes move from `internal/api/router.go` into the owning module's `Routes`:

  | Context | Routes |
  |---|---|
  | `applications` | `/applications/**`, `/application-statuses/**` |
  | `identity` | `/auth/**`, `/google/**`, `/profile`, `/ai-credentials`, `/ai-prefs` |
  | `cvtemplates` | `/cv-templates/**`, `/tracked-docs/**` |
  | `scoring` | `/scoring-config`, `/scoring-options`, `/scores/**` |
  | `jobsearch` | `/jobs/**`, `/companies/**`, `/source-targets/**`, `/sources/**`, `/ingest` |

- **Schema.** No table changes, except that `google_oauth_tokens` gets mirrored into `schema.sql`. Ownership:

  | Context | Tables it writes |
  |---|---|
  | `jobsearch` | `jobs`, `job_urls`, `companies`, `company_boards`, `board_poll_state`, `board_job_observations`, `job_candidates`, `candidate_discoveries`, `candidate_assessments`, `harvest_runs`, `source_targets`, `tracked_companies` |
  | `scoring` | `search_config`, `scoring_options`, `option_answers`, `job_scores`, `effect_outbox` |
  | `applications` | `applications`, `application_statuses` |
  | `cvtemplates` | `tracked_docs`, `tracked_doc_tabs` |
  | `identity` | `users`, `sessions`, `user_ai_credentials`, `google_oauth_tokens` |

  `notification_digests` is unreferenced. Leave it alone and don't assign it.

- **Module shape.**
  - `internal/<ctx>/module.go`: `New(deps...) *Module` and the facade.
  - `internal/<ctx>/routes.go`:
    - `Routes(chi.Router)` is mounted inside the session-protected group.
    - `PublicRoutes(chi.Router)` is optional and mounted outside it (identity's login, signup and OAuth start; jobsearch's ingest behind the service token).
  - `internal/<ctx>/internal/<feature>/` holds services.
  - `internal/<ctx>/internal/store/` holds the store, `queries/` and generated `sqlc/`.
- **Shared kernel.** `internal/handlers` owns `Handle`, the CRUD generics and the session context key:
  - `WithSession(ctx, dto.Session) context.Context`
  - `UserID(*http.Request) (string, error)`
  - `DecodeBody[T]`, `DecodeQuery[Q]`, `WriteJSON`

  `internal/pgtest` provides `New(testing.TB) *pgxpool.Pool`: one container per test binary, migrations run from the repo root, and truncation between tests.
- **Facade contracts** (bounded to what other contexts, the worker or admin actually call).
  - `applications`: `SeedDefaults(ctx, tx pgx.Tx, userID string) error`. It's pool-based `SeedDefaults(ctx, userID)` until Phase 3.
  - `identity`:
    - `Middleware() func(http.Handler) http.Handler`
    - `Credential(ctx, userID, provider) (string, error)`
    - `Profile(ctx, userID) (dto.Profile, error)`
    - `DocsClient(ctx, userID) (DocsClient, error)`
    - `DeleteExpiredSessions(ctx) error`
    - `CreateUser(ctx, dto.CreateUserInput) (dto.User, error)`
  - `scoring`:
    - `Run(ctx) error`, which blocks and drains answer effects
    - `SearchConfig(ctx, userID) (dto.SearchConfig, error)`
    - `OpsState(ctx) (dto.OpsState, error)`
    - `AddOption`, `RewordOption`, `RetireOption`
    - transaction ports:

      | Port | Called by |
      |---|---|
      | `JobsChanged(ctx, tx, jobIDs []string) error`: drop stale answers and queue answer effects | job save/upsert |
      | `JobsClosed(ctx, tx, jobIDs []string) error`: drop answers | closing jobs, board completion |
      | `CompanyTracked(ctx, tx, userID, companyID string) error`: queue scores for known open jobs, with no alerts | tracking a company |

  - `jobsearch`: grouped sub-facades, so the worker sees three small interfaces rather than one wide one:
    - `m.Boards()`: due/active lists, claim/complete/fail, verify, verified-board lookup
    - `m.Targets()`: recoverable targets, claim, run transitions, get
    - `m.Catalog()`: new-URL filter, company upsert/crawl, last-scraped, candidate save, expired-candidate cleanup

    Plus `Reconsider(ctx, userID) error`, which scoringconfig calls when criteria change.

- **Consumer-side interfaces.** Each caller declares the narrow interface it needs in its own package, and a `*Module` satisfies it. Nothing imports another context's internals.
- **Writes are owned, reads may join.** A store may SELECT-join any table. It may only INSERT/UPDATE/DELETE its own. A write that has side effects in another context calls that context's transaction port inside the caller's transaction.
- **Errors.** A store declares its own sentinels (`ErrNotFound` plus specific ones), kinded with `apperr`. The root re-exports only the ones used across contexts.
- **Tests.** Services use hand fakes of their own `store` interface. Stores use `pgtest`. The router table test stays and covers every mounted route.
- **Enforcement.** Each moved context gets a depguard rule denying `internal/data/providers`, `internal/data/db` and `internal/api`. sqlc gets one block per context.
- **Composition.** `cmd/api/main.go` builds the pool, the queue and every module, then passes the modules to `api.NewRouter(modules...)`. `cmd/worker` and `cmd/admin` build only the modules they call.
- **Auth.** This is unchanged: the session cookie goes through identity's middleware, then `handlers.UserID`. The ingest bearer token (`INGEST_SERVICE_TOKEN`) goes through `ServiceTokenMiddleware`, mounted via jobsearch's `PublicRoutes`.

---

## Phase 1: Shared handler kernel

**User stories**: every existing route keeps its status codes, error bodies and auth behaviour.

### What to build

Move `Handle`, the CRUD generics and the adapter helpers into `internal/handlers`, together with the session context key. The session middleware sets the key through `handlers.WithSession`. Export `UserID`, `DecodeBody`, `DecodeQuery` and `WriteJSON`. The misfit handlers (auth, google, CV export, ingest) stay in their current package and import the kernel. The router imports both packages.

### Acceptance criteria

- [ ] `internal/handlers` imports nothing under `internal/api`. The shared depguard rule passes.
- [ ] `generic_test.go` and the router table test pass unchanged.
- [ ] No route path, status or error body changes. The e2e suite passes.

---

## Phase 2: `applications` tracer

**User stories**:

- a user lists, creates, updates and deletes Applications and Statuses
- a new user gets default Statuses on signup

### What to build

This is the first full context:

- `internal/pgtest`
- `api.NewRouter(modules...)` mounting `Routes`
- a sqlc block for `applications`
- store, services and module

The application list keeps its read-join onto `jobs`. Legacy signup calls `SeedDefaults` (the pool-based variant) through a local `StatusSeeder` interface. Delete the application providers, their mocks, the `*DB` methods and the legacy queries. Add the depguard rule. Update `AGENTS.md` § Migration status. Finish with a retro, and fix the ADR or skills where the pilot proved them wrong.

### Acceptance criteria

- [ ] `sqlc generate && git diff --exit-code` is clean with two sql blocks.
- [ ] Store tests run on `pgtest`, and service tests use hand fakes. No `providers.Mock*` is referenced.
- [ ] `internal/applications` imports no `providers`, `data/db` or `internal/api` (depguard).
- [ ] Manual check: signing up shows the default Statuses, and application CRUD works in the UI.
- [ ] The ADR, skills and `AGENTS.md` reflect the retro.

---

## Phase 3: `identity` core

**User stories**:

- a user signs up, logs in, logs out and fetches `me`
- signup creates the user and their default Statuses atomically
- expired sessions are cleaned up

### What to build

Move users, sessions, the auth service, the session middleware and the cookie misfit handlers into `identity`. Its `PublicRoutes` carries login and signup. Signup runs in one transaction and calls `applications.SeedDefaults(ctx, tx, userID)`. The router gets its session middleware from `identity.Middleware()`. The worker's session cleanup and admin's `CreateUser` switch to the identity facade.

### Acceptance criteria

- [ ] A signup whose seeding fails leaves no user row (store test).
- [ ] The auth routes behave the same: cookies, 401s, e2e login.
- [ ] Worker and admin compile against the facade, with no `*jobsdb.DB` calls for identity tables.

---

## Phase 4: `identity` connections

**User stories**:

- a user connects, checks and disconnects Google
- a user manages AI credentials and prefs
- a user edits their profile

### What to build

Move into `identity`:

- the Google Link: OAuth start/callback misfits, the token store, and the Docs client
- AI credentials and credstore
- aiprefs and profile

Mirror `google_oauth_tokens` into `schema.sql` and move its raw SQL into sqlc queries. Expose `Credential`, `Profile` and `DocsClient`. Legacy scoring and cvtemplates consume them through local interfaces.

### Acceptance criteria

- [ ] `google_oauth_tokens` is generated by sqlc. No raw SQL remains.
- [ ] The OAuth round-trip works by hand, and the Google status and disconnect routes are unchanged.
- [ ] Legacy suitability gets credentials and profile through identity's facade.

---

## Phase 5: `cvtemplates`

**User stories**: a user lists Tracked Docs and CVs, hides Tabs, and exports a CV as PDF.

### What to build

Move tracked docs, tabs, the CV service and the PDF streaming misfit into `cvtemplates`. It reaches Google only through `identity.DocsClient`.

### Acceptance criteria

- [ ] cvtemplates imports no Google token code.
- [ ] CV list, visibility and PDF export work by hand. The e2e suite passes.

---

## Phase 6: `scoring` move

**User stories**:

- jobs get a Suitability score for interested Users
- changing Picks rescores for free
- new-Job alerts fire at the User's threshold
- the "Scoring stalled" alert still has data

### What to build

Move into `scoring`:

- search config, options, answers, scores and the outbox
- suitability and scoringconfig
- Jev and notify, including the templates path and the `just build-emails` output

`scoring.Run` replaces the ticker in `main.go`. `telemetry.StateCollector` takes an `OpsState` interface. Admin's option commands and the worker's reject filter use the facade. Legacy jobsearch still writes scoring tables in this phase, through its old queries, which is a known, temporary exception. Phase 7 removes it.

### Acceptance criteria

- [ ] `main.go` has no scoring loop. Stopping the context stops `Run`.
- [ ] Recompute-on-save is covered by the existing suitability tests, which pass on hand fakes and `pgtest`.
- [ ] The `/metrics` `jobscraper_*` series are unchanged.

---

## Phase 7: Scoring transaction ports

**User stories**:

- a Job edit re-queues its answers atomically
- a closed Job drops its answers
- tracking a Company scores its open Jobs without alerts

### What to build

Implement `JobsChanged`, `JobsClosed` and `CompanyTracked` in scoring. Switch the legacy jobsearch transactions (canonical save, board completion, marking jobs closed, company tracking) to call them, and remove every jobsearch query that writes `option_answers` or `effect_outbox`.

### Acceptance criteria

- [ ] No query outside scoring writes a scoring table (checked with grep in review, and by sqlc package separation once Phase 8 lands).
- [ ] Rolling back a job save also rolls back its queued effect (store test).
- [ ] The existing canonical-save, board-poll and tracking tests pass.

---

## Phase 8: `jobsearch` API side

**User stories**:

- a user pages Jobs sorted by Suitability
- a user manages Companies, Boards and Source Targets
- the worker's `POST /ingest` saves Jobs

### What to build

Move into `jobsearch`:

- jobs, companies, source targets and sources
- candidates and ingest
- the remaining `internal/data/db` tables

Its `PublicRoutes` carries ingest behind the service token. The score-sorted job page stays one read-join onto `job_scores`. Expose `Reconsider` to scoringconfig and the `Boards`/`Targets`/`Catalog` sub-facades.

### Acceptance criteria

- [ ] The job page ordering and cursors are identical (existing page tests).
- [ ] Ingest accepts only the service token, and a repeated delivery keeps one canonical Job.
- [ ] `internal/api/router.go` contains only middleware, CORS and module mounting.

---

## Phase 9: Worker onto `jobsearch`

**User stories**: board polling, discovery runs, company harvest and cleanup keep working.

### What to build

`cmd/worker` stops building `*jobsdb.DB`. It builds the pool, then `jobsearch`, `scoring` and `identity`. It passes the `Boards`/`Targets`/`Catalog` sub-facades, `scoring.SearchConfig` and `identity.DeleteExpiredSessions` to worker packages through their existing narrow interfaces.

### Acceptance criteria

- [ ] `cmd/worker` doesn't import `internal/data/db`.
- [ ] Manual check with `just up`: a scrape-now run succeeds, a due board is polled, and Jobs arrive via ingest.
- [ ] The worker's `/metrics` output is unchanged.

---

## Phase 10: Cleanup

**User stories**: none. This removes the legacy layout.

### What to build

Delete:

- `internal/data/providers`, `internal/data/db`
- `internal/api/services`, `internal/api/handlers`
- `newServices`
- the legacy sqlc block

Also:

- Collapse the per-context depguard rules into one glob.
- Remove the legacy notes and Migration status from `AGENTS.md`, `new-handler` and `schema-change`.
- Delete or rewrite `docs/architecture/system-overview.md`.

### Acceptance criteria

- [ ] `grep -r "data/providers\|data/db" internal cmd` finds nothing.
- [ ] `just lint`, `go test ./...` and the e2e suite pass.
- [ ] The docs describe only the context layout.
