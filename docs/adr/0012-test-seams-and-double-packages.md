# ADR 0012 — Test seams: `Build(Deps)`, `*test` double packages and store contract suites

Tests were written for whatever code was in front of the agent, so it was unclear what each proved, and nothing kept a service's hand fake in step with the real store. The policy is in the `testing-policy` skill; this records the layout changes it needs. Background: `docs/research/go-testing-synthesis.md`.

- **Unit.** The behaviour a package's exported surface offers. Unexported helpers are tested only through that surface.
- **`<ctx>test` packages.** `internal/services/<ctx>/<ctx>test` exports one stateful, map-backed `NewFakeStore()` of the context's whole `Store` and `RunStoreContract(t, newStore)`. The service's tests use the fake; the context's `store_test.go` runs the same contract against the real store on `pgtest`, so the fake can't drift. This follows `testing/fstest` (`MapFS` beside `TestFS`) and Google's `<pkg>test` convention. The package also exports behaviour-named stubs of the context's facade (`scoringtest.ScoresAs(80)`), for other contexts' tests. A split-out feature package (`sourcetargets`) is served by its context's fake. Where the real store enforces something the fake skips (idempotent upserts, cascading deletes, claim rules), fix the fake and add a contract case.
- **`Build(Deps)` seam.** Each module exports `Build(Deps) *Module`, taking store interfaces and collaborators. `New(pool, …)` becomes a wrapper that builds the real stores and calls `Build`. Route and facade tests call `Build` with fakes; `cmd/api` keeps calling `New`.
- **Enforcement.** Production code may not import a `*test` package (depguard). Store tests may. `forbidigo` bans `reflect.DeepEqual`, `time.Sleep` and testify in `_test.go`.
- **Store tests** run the contract suite and otherwise cover only what Postgres proves: constraints, `ON CONFLICT`, cross-context joins, tx-port rollback, concurrent writes. They stay serial on pgtest's truncated shared DB.

Amends ADR 0008's and ADR 0011's tests bullets: route tests go through the module's real `Routes` and middleware over `Build(fakes)`, using shared `handlerstest` checks instead of a per-handler recorder test.

Rejected alternatives:

- Real Postgres in service tests: every service test becomes Docker-bound and slow for fidelity the contract suite already provides.
- Stub fakes with no contract: they drift and prove nothing about the store.
- Contract suites duplicated on each side by convention: nothing mechanically stops drift.
- An unexported test-only module constructor: forces white-box route tests.
- Per-test database clones or `t.Parallel()` store tests: six store test files don't justify rewriting pgtest yet.
- A CI coverage floor: it rewards the per-file tests this policy exists to stop.

Trade-off: each context gains up to two exported test packages and each module a second constructor. Contract suites must be kept in step with store interfaces by hand.

Status: accepted (2026-09-28). Pilot in `applications`; other contexts migrate when next touched.
