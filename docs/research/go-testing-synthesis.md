# Go testing: what to test and how — research synthesis

_Last updated: 2026-09-28. Background research only; not a policy._

## 1. Use-case frame

job-scraper is a Go modular monolith split into five contexts. Each context has feature services, sqlc stores on
Postgres, thin generic HTTP handlers and a facade other contexts call. A RabbitMQ worker runs scraper adapters. Today,
tests get written for whatever code is in front of the agent, so it's unclear what each one proves. The question is
which kinds of runtime code deserve which kind of test, and what a good test of each kind looks like, so that a new
module's test set is predictable before anyone writes it.

## 2. The mental model

1. **Test behaviour, not structure.** A test should fail when observable behaviour changes and survive a refactor that
   keeps it the same. Beck calls these properties *behavioural* and *structure-insensitive*; Google calls them
   *unchanging tests*. This is the main filter for which tests are worth keeping.
2. **A unit is a unit of behaviour exposed on a public surface.** It is not a function or a file. Google's rule is that
   a helper used by one or two callers gets no direct tests; anything built for outside callers does. In Go terms, the
   unit is the package's exported surface, and here that means the service methods and the module facade.
3. **Solitary vs sociable tests.** Solitary tests replace every collaborator; sociable tests use real ones and replace
   only the slow or external. Fowler shows that "pyramid vs honeycomb" is mostly an argument over this word. Define
   "unit" and "integration" in your own terms and the shape debate largely goes away.
4. **Test sizes beat test labels.** Google sizes tests by resources, not by name. A small test runs in one process
   with no I/O. A medium test is on one machine and may use Docker (your testcontainers store tests). A large test
   spans processes or machines. Sizes tell you where a test can run and how flaky it may be.
5. **Managed vs unmanaged dependencies** (Khorikov). A managed dependency is reachable only through your app, like
   your Postgres and your queue, so how you talk to it is an implementation detail: use the real thing. An unmanaged
   dependency is observable by others, like job boards, ATS APIs, email, OAuth and outgoing messages. How you talk to
   it is a contract: replace it at the edge.
6. **Fidelity order for doubles: real > fake > stub > mock.** A fake is a working lightweight implementation. A stub
   returns canned answers. A mock records and verifies calls. Each step down makes the test less like production.
   Google's 2024 guidance: take the most fidelity you can *without increasing the test's size*.
7. **Contract (conformance) tests keep doubles honest.** One test suite runs against both the real implementation and
   the fake, so the fake can't silently drift. Fowler uses the same term for scheduled checks against third-party
   services.
8. **Change-detector tests have negative value.** A test that restates the code's call graph ("calls A, then B") fails
   on every refactor and never on a real bug. Rewrite it at a higher level or delete it.

## 3. Consensus view

- **Go through the public surface.** Needing to test an unexported function is a design smell. Use black-box `_test`
  packages where practical.
- **Assert on state and outputs, not interactions.** Interaction checks are allowed only for state-changing calls
  across a system boundary ("was the email sent", "was the message published"), never for queries.
- **Use a real database for code that owns SQL.** Every source that discusses it agrees: Postgres in Docker, pinned to
  the production version. The goal is to check that the adapter uses the database correctly, not that Postgres works.
- **Hand-written doubles over generated mocks,** built on small interfaces declared by the consumer. Bourgon, Three
  Dots and Google's history all point here. Google warns that a hand-rolled double can still be low-fidelity; what
  matters is how closely it behaves like the real thing, not whether a framework wrote it.
- **Standard library plus `go-cmp`, no assert frameworks** (Go wiki, Google, Bourgon). Diff whole values with
  `cmp.Diff`. Write failures as `Func(in) = got, want want`. Match errors with `errors.Is`/`errors.As`, never by
  string.
- **Use table tests when every case is checked the same way.** If cases need different assertions (`if tc.wantErr`
  branches), write separate tests. Name every case and run each in `t.Run`.
- **Keep E2E and integrated tests few.** They show that the pieces connect, not that the logic is right. When a
  high-level test catches a bug no low-level test caught, add the low-level test.
- **Determinism is non-negotiable.** No `time.Sleep`: poll, or use `Eventually`. Flaky tests are treated as broken.
- **Duplication in tests is fine** if it keeps each test readable on its own (Google's "DAMP over DRY").

## 4. Tensions and trade-offs

- **Suite shape.** Vocke and Google prefer many small tests (about 80% unit). Spotify prefers mostly service-level
  integration tests and few implementation-detail tests. Three Dots sits between: unit tests for domain logic and
  parsers, integration tests for adapters, a few component tests over HTTP. Fowler: once "unit" is defined, these mostly
  agree. The real decision is how sociable your service tests are.
- **Handlers: isolate or go end to end?** Bourgon and Google test each piece in isolation with fakes. Ryer calls the
  real `run()`/router with middleware, and deletes tests that only repeat an end-to-end test, arguing `httptest`
  recorder tests skip auth and middleware. With your thin generic handlers (ADR 0008), Ryer's argument is the stronger
  one.
- **Faking a database your app owns.** Google allows it if the fake is contract-tested against the real store.
  Khorikov says a managed dependency should always be real. Your current split (hand fake in service tests, real
  Postgres in store tests) is acceptable to Khorikov only if the store tests carry the database fidelity. Google adds:
  and the fake must pass the store's contract suite.
- **Isolating Postgres state between tests.** There are four options:
  - A shared DB where each test writes unique data (Three Dots). Cheapest and parallel, but the test can't assume
    empty tables.
  - A per-test clone from a template (pgtestdb, or testcontainers `Snapshot`/`Restore`). About 10–20ms per clone, and
    the test gets a real empty database.
  - A schema per test, reusing cleaned schemas. Brandur measured this about 3.5× faster than cloning.
  - Roll back a transaction after each test. Fastest, but breaks when the code commits itself, uses several
    connections, involves workers or LISTEN/NOTIFY, or depends on sequences.
- **Parallelism.** Ryer, Google and Three Dots use `t.Parallel()`. Hashimoto (2017) said don't; run several processes
  instead. Modern practice has moved to parallel, which only works if tests don't share state.
- **Where doubles live.** Google puts a reusable double in a sibling `<pkg>test` package, named by behaviour
  (`AlwaysDeclines`). Your repo keeps fakes inside the consuming package. The dividing line is reuse: a fake of one
  service's store stays local; a fake of a module facade used by several contexts belongs in `<ctx>test`.
- **Minor style splits.** Cheney keys table cases with a map, which gives random order and exposes coupling between
  cases. Google uses a slice with named fields. Either works; pick one.

## 5. Practical patterns for this codebase

Ordered by how directly they map onto the code you already have.

**Services with store interfaces: small, sociable-within-the-package tests.** Test through the service's exported
methods, with a stateful in-memory fake of its store (a map plus the sentinels the real store returns), not a stub
that returns canned rows. Assert the returned DTO and the `apperr` kind; don't assert which store methods were called.

**Stores: medium tests against real Postgres, plus a contract suite shared with the fake.** Three Dots runs one suite
against every implementation of the repository interface. That same move proves the service's hand fake matches the
sqlc store:

```go
func runStoreContract(t *testing.T, newStore func(t *testing.T) Store) {
    t.Run("get missing returns ErrNotFound", func(t *testing.T) {
        _, err := newStore(t).Get(ctx, uuid.New())
        if !errors.Is(err, ErrNotFound) { t.Fatalf("Get(missing) err = %v, want ErrNotFound", err) }
    })
}
// store_test.go:   runStoreContract(t, func(t *testing.T) Store { return store.New(pgtest.DB(t)) })
// fake_test.go:    runStoreContract(t, func(t *testing.T) Store { return newFakeStore() })
```

Store tests should target what only the database can prove: constraints, uniqueness, `ON CONFLICT`, joins across
contexts, transaction rollback and concurrent writes. Three Dots races 20 goroutines and asserts exactly one wins. They
also recommend *test sabotage*: remove the rollback and confirm a test goes red. Generated sqlc code is covered through
these tests, never tested directly.

**Handlers and routes: few, through the real router.** Given ADR 0008, per-handler recorder tests mostly restate the
generics. The sources point to a small number of route-level tests through the real mux and middleware (auth, decode
errors, status mapping). The exceptions are the hand-built `Handle` routes (cookies, OAuth, streaming, ingest), which
contain their own logic. Compare JSON bodies by unmarshalling both sides, never byte for byte.

**Scraper adapters: golden files, plus fuzzing the parsers.** Hashimoto's `-update` flag pattern is what your
`just cli rebase` does. Google calls this A/B diff testing: cheap to write, but the diffs must be normalised and
approved on purpose. Parsers take untrusted input, which makes them the best fuzz targets in the repo:

```go
func FuzzParse(f *testing.F) {
    f.Add(mustRead(f, "testdata/listing.html"))
    f.Fuzz(func(t *testing.T, b []byte) { _, _ = Parse(b) }) // invariant: never panics
}
```

Plain `go test` runs the seed corpus as regression tests. Failing inputs land in `testdata/fuzz/` and should be
committed. Checking adapters against the live boards is a contract test in Fowler's sense: run it on a schedule, not
as a merge gate.

**Unmanaged edges (job boards, OAuth, email, outgoing queue messages).** Use a real network connection to an
`httptest.NewServer` serving recorded fixtures, rather than a fake `http.Client` (Google, Hashimoto). For outgoing
messages, asserting "published X" is legitimate interaction testing: it is a state-changing call across a boundary.

**Whole-app component tests: a handful.** Three Dots and Ryer both boot the real service (a real DB, only other
systems replaced) and run happy paths over HTTP. Corner cases stay in lower-level tests. Ryer's
`run(ctx, args, getenv, stdin, stdout, stderr)` signature plus `waitForReady` polling `/healthz` makes this parallel-safe.

**Test helpers.** A helper takes `t`, calls `t.Helper()`, fails with `t.Fatal` on setup errors and cleans up with
`t.Cleanup`. Expensive shared setup (one container per binary) belongs in `TestMain`, which is what `pgtest` does.
Helpers that also assert are discouraged: return an `error` or a `cmp.Option`, and fail in the test itself.

## 6. What to watch out for

- **Stub-style fakes drift.** A fake that returns whatever the test primes it with proves nothing about the real
  store. Make it stateful and put it under the contract suite, or keep it too small to drift.
- **Mock-order assertions on orchestration code** (`module.go` wiring, generic handlers) are change detectors. Cover
  them from above.
- **Don't assert list lengths in shared-DB tests.** Look up the rows your test created. Length checks break the moment
  another parallel test writes to the same table.
- **Never call `t.Fatal` from a goroutine other than the test's own.** Use `t.Error` and return.
- **testcontainers `Restore` drops the connected database.** Don't name it `postgres`. Use `WithSQLDriver("pgx")` to
  avoid the slow `docker exec` path.
- **Golden files hide noise.** Timestamps, ordering and IDs must be normalised before comparing, or rebases become
  rubber stamps.
- **Every larger test needs an owner** (Google), or it rots into a skipped test.

## 7. Gaps in the reading list

- **Testing across contexts inside one binary.** No source covers facade contracts between contexts in the same
  process, or tx-scoped ports like `scoring.JobsChanged(ctx, tx, …)`. Three Dots' component tests cover services
  talking over the network, not in-process module seams. How to test that a facade honours its promise to other
  contexts is unresolved.
- **Queue consumers.** Brandur mentions the API-publishes, worker-completes flow as a reason for per-test databases.
  Nothing covers RabbitMQ delivery semantics (redelivery, idempotency, poison messages) or what a consumer test should
  assert.
- **sqlc specifically.** No source addresses generated-query code. "Covered through the store" is an inference from
  the adapter principles, not something any source states.
- **Making a test's purpose obvious.** This is your core complaint. Sources say "test behaviours, not methods" and
  "readable: shows why the test exists", but none gives a naming or grouping convention a reviewer could check.
- **Coverage targets.** Only Three Dots names a number (70–80%). Nobody addresses coverage per code type.
- **Agent-generated tests.** None of the sources address stopping an agent from writing a test for every file.
  Mapping code type to test type is your own policy decision.
- **Frontend.** Out of scope for this reading list.

## 8. Recommended next steps

- **Classify the existing tests before writing policy.** Tag every `_test.go` by code type (service, store, facade,
  handler, adapter, worker) and by kind (small, medium, golden, component, change-detector). Keep the numbers; they
  show where the tests are today.
- **Decide the one-line definitions of "unit" and "integration" for this repo.** Fowler's point is that most of the
  argument goes away once these are pinned. Every later rule depends on them.
- **Pilot a store contract suite in one context,** run against both the pgtest store and the service's hand fake. It
  is the highest-leverage pattern here, and it will expose whether the current fakes have drifted.
- **Pick a Postgres isolation strategy on purpose:** unique-data-per-test, a pgtestdb clone, or `Snapshot`/`Restore`.
  Decide before parallelising store tests. Measure it on your own migrations.
- **List the change-detector tests you'd delete.** Deletions are the fastest proof the policy works, and they show
  what the approach doc needs to forbid.

## Sources

- Go wiki, TestComments — https://go.dev/wiki/TestComments
- Google Go Style Guide, tests — https://google.github.io/styleguide/go/best-practices#tests
- Dave Cheney, Prefer table driven tests — https://dave.cheney.net/2019/05/07/prefer-table-driven-tests
- Peter Bourgon, Go best practices six years in — https://peter.bourgon.org/go-best-practices-2016/#testing
- Mat Ryer, How I write HTTP services in Go after 13 years — https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/
- Mitchell Hashimoto, Advanced Testing with Go — https://speakerdeck.com/mitchellh/advanced-testing-with-go
- Go docs, Fuzzing — https://go.dev/doc/security/fuzz/
- Software Engineering at Google, ch. 12–14 — https://abseil.io/resources/swe-book/html/ch12.html
- Google Testing Blog, Increase test fidelity by avoiding mocks — https://testing.googleblog.com/2024/02/increase-test-fidelity-by-avoiding-mocks.html
- Google Testing Blog, Change-detector tests considered harmful — https://testing.googleblog.com/2015/01/testing-on-toilet-change-detector-tests.html
- Vladimir Khorikov, When to mock — https://enterprisecraftsmanship.com/posts/when-to-mock/
- Kent Beck, Test Desiderata — https://testdesiderata.com
- Three Dots Labs, Microservices test architecture — https://threedots.tech/post/microservices-test-architecture/
- Three Dots Labs, Database integration testing — https://threedots.tech/post/database-integration-testing/
- Ham Vocke, The Practical Test Pyramid — https://martinfowler.com/articles/practical-test-pyramid.html
- Martin Fowler, On the diverse and fantastical shapes of testing — https://martinfowler.com/articles/2021-test-shapes.html
- Spotify, Testing of microservices — https://engineering.atspotify.com/2018/01/testing-of-microservices
- testcontainers-go Postgres module — https://golang.testcontainers.org/modules/postgres/
- pgtestdb — https://github.com/peterldowns/pgtestdb
- Brandur, pgtestdb — https://brandur.org/fragments/pgtestdb
- Martin Fowler, ContractTest — https://martinfowler.com/bliki/ContractTest.html
