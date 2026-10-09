# ADR 0025 — One shared, self-destructing test database container

## Context

`pgtest` started one Postgres container per test binary and relied on Ryuk to remove it. A `go test ./...` started one container per package, and several agent sessions running at once multiplied that. Ryuk missed containers whenever a run was killed, and each container left an anonymous volume behind. On 2026-10-09 the machine had 194 containers and 186 volumes, and Docker was using most of its memory and CPU ([#840](https://github.com/O-Marsters-1997/job-scraper/issues/840)).

## Decision

- **One container for the whole machine.** Every test binary, worktree and session reuses `job-scraper-pgtest-v1` (`postgres:17-alpine`, `fsync=off`, `max_connections=500`) through testcontainers' `WithReuseByName`. Reuse doesn't compare config, so a config change bumps the name's version suffix.
- **Ryuk is off for pgtest.** In testcontainers v0.41 a reused container keeps its creator's Ryuk session, so Ryuk would kill the shared container when the first binary exits. The only switch is the process-wide `TESTCONTAINERS_RYUK_DISABLED`, which pgtest sets before its first container. Other tests, such as the RabbitMQ broker test, keep Ryuk.
- **The container stops itself.** A watchdog in the entrypoint stops Postgres after 5 minutes with no client connections. The container runs with AutoRemove and keeps `PGDATA` on tmpfs, so stopping it leaves no container and no volume, even after a SIGKILLed run.
- **One database per test binary.** Each binary gets `pgtest_<unix>_<rand>`, cloned from `pgtest_tpl_<migrations hash>`, so packages keep running in parallel. Migrations run once per hash. The template is built under a different name and renamed when it's done, all under a Postgres advisory lock, so a crashed build never gets published. Each pool keeps one connection open, which shows the watchdog and the sweep that the binary is alive.
- **Orphan sweep.** At start, every binary drops `pgtest_*` databases that are over a minute old and have no connections. A busy machine may never go idle for 5 minutes, and the sweep stops dead databases from piling up in the meantime.
- **`just clean`** is for what tests don't own: compose stacks of finished worktrees (`fleet-N`, `job-scraper-<slug>`, `agent-*`) and their volumes, stopped testcontainers, and dangling anonymous volumes. It never touches the dev `job-scraper_*` stack or other projects. `.treepad.toml` is tracked, and its `post_remove` hook runs `just clean`, so removing a worktree tears down its stack.

## Consequences

- A full `go test ./...` runs one Postgres container no matter how many packages or sessions run at once, and the first run after an idle spell pays for one container start and one migration.
- Postgres tests no longer get a fresh server, only a fresh database. Anything server-wide (roles, `pg_stat_activity`, advisory locks) is shared across binaries, so filter by `current_database()`.
- The RabbitMQ broker test keeps Ryuk and a real disk, because it restarts the broker and expects messages to survive.
