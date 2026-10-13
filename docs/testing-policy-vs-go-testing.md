# `testing-policy` and `pgtest` against the global `go-testing` skill

Compared on 2026-10-13 for [#859](https://github.com/O-Marsters-1997/job-scraper/issues/859) against
`go-testing` from O-Marsters-1997/my-claude-code#210. The ADRs are not amended here: #874 owns
[ADR 0012](adr/0012-test-seams-and-double-packages.md) and
[ADR 0025](adr/0025-shared-self-destructing-test-database.md) (`0025-shared-self-destructing-test-database.md`;
there are two `0025-*` files until #874 renumbers one).

## Verdicts

| Area | Global `go-testing` | This repo | Verdict |
|---|---|---|---|
| Assertions, tables, `t.Helper`, `t.Cleanup`, no `time.Sleep`, `errors.Is` | Owns them | `testing-policy` pointed at `go-idiomatic` for them | Adopt global. The skill now points at `go-testing` and restates none of it |
| Double fidelity (real, fake, stub, mock) | Real > fake > stub > mock; mock only state-changing calls across a boundary, by hand-written recorder | Same order; no generated mocks, no testify; interaction assertions only across unmanaged boundaries (ADR 0012) | Agree. The repo's boundary list (messages, email, LLM, retried HTTP) is a worked instance, so it stays |
| Fake plus contract suite | One contract suite runs against fake and real; error paths via a one-method override | `RunStoreContract` in `<ctx>test`, one-method override | Agree |
| Black-box `package foo_test` | Rule 1 | Facade and route tests use `<ctx>_test`; ADR 0012 rejects white-box route tests | Agree |
| `cmp.Diff`, no `reflect.DeepEqual` | Rule 6 | go-cmp is in `go.mod`; `forbidigo` bans `reflect.DeepEqual` and `time.Sleep` in tests | Agree. Lint enforces it |
| Test dependencies | go-cmp only | No testify, no generated mocks | Agree |
| Compare JSON by decoding | Rule 13 | Route tests decode both sides | Agree |
| Fuzz parsers, commit `testdata/fuzz/` | Rule 14 | `FuzzParse` per HTML source | Agree |
| Test kinds, sizes, never-test list, file naming, queue-consumer cases, source adapters | Out of scope: "what to test is the repo's `testing-policy`" | The map | Keep as repo policy; this is the part the global skill delegates |
| Container lifecycle: lazy `sync.Once`, Ryuk off, idle-stop entrypoint, AutoRemove, tmpfs, `WithReuseByName` | Same design | Same design (ADR 0025) | Agree |
| `TEST_DATABASE_URL` shared server | Supported | Not supported | Adopt global in the migration ticket (CI, developers with their own server) |
| Retry the container start (2 attempts) | Yes | No | Adopt global in the migration ticket |
| Template database, migrated once per migrations hash, cloned | Yes | Yes (`pgtest_tpl_<hash>`, advisory lock, rename on completion) | Agree |
| Database granularity | One database per test, dropped on cleanup | One per test binary; every table `TRUNCATE ... CASCADE`d in `New` | Adopt global (below) |
| `t.Parallel()` in DB tests | Safe, a database is never shared | Banned in any test that calls `pgtest.New` | Adopt global (below) |
| `New` return type | DSN string; `Create` returns DSN plus drop func | `*pgxpool.Pool`, plus `Pool()` and `InTx` | Keep `InTx`: the global helper has no equivalent. Pool-vs-DSN moves with the migration |
| Orphan sweep | Drops stale `pgtest_*` databases | Same, at start of each binary | Agree |

## Decision: truncation and serial store tests

Truncation plus serial store tests is superseded. #874 (W0-03) settles that every DB test gets its own
database cloned from the template and runs with `t.Parallel`, with no `TRUNCATE`. #859 records that
decision and does not change `pgtest`; migrating it is a follow-up ticket (number in the PR body).

ADR 0012 rejected per-test clones because "six store test files don't justify rewriting pgtest yet".
`pgtest` already creates and clones a database per binary, so per-test is a small step, and the
`TRUNCATE` costs are visible: serial store tests, `InTx` existing partly so a failed test can't leave
locks that block the next `TRUNCATE`, and `New` needing a table list.

Until that ticket lands, `testing-policy` describes truncation and serial store tests as the current
behaviour, so an agent writing a store test today does not use `t.Parallel()` against a shared database.

## Deviations that stay as repo policy

- The test map, never-test list, naming rules, queue-consumer cases and source-adapter rules.
- `<ctx>test` double packages and `RunStoreContract` (ADR 0012).
- `pgtest.InTx`, until the migration ticket decides otherwise.
- The container name `job-scraper-pgtest-v1`. The global helper uses the same mechanism with its own
  name, so this is not a deviation in kind.

## Dangling reference

`testing-policy` and ADR 0012 cite `docs/research/go-testing-synthesis.md`, which is not in the repo.
`testing-policy` drops the citation; ADR 0012's is left to #874.

## Note for my-claude-code#210 (not posted)

`WithReuseByName` feeding back into the global template: the global helper already carries it, with
Ryuk off and the idle-stop entrypoint, matching this repo. What this repo has that the template lacks is
`InTx` (rollback registered as cleanup, so a `t.Fatal` leaves no locks). One caveat worth stating in the
template: reuse doesn't compare container config, so any config change needs a bump of the container
name's version suffix, as ADR 0025 does with `-v1`.
