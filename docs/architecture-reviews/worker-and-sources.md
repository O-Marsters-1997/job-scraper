# Architecture Review — Worker & Sources

**Scope:** the worker implementation, how new scraping sources plug in, and how
reusable the shared scraping code is. Focused review (three lenses: extensibility,
modularity/reuse, worker simplicity & failure modes), not a full six-dimension sweep.

**Date:** 2026-07-01 · **Branch:** `chore/architectural-audit`

---

## Verdict

The design is **sound and pragmatic**. The `Source` interface + `PaginatedBase`
split is genuinely good: a source author writes fetch-and-map logic and inherits
HTTP, proxy, headers, jitter, and pagination. Adding an ATS source is ~100 lines,
adding an HTML source ~300. Nothing here needs a rewrite.

The real issues are not correctness — they are a **silent-failure trap when adding
a source** and **two observability blind spots in the worker**. Fix those; leave the
rest alone until it hurts.

Three cross-cutting themes, ranked by impact:

1. **The builder is an un-validated coordination point** — the one place "add a source" goes wrong silently.
2. **The worker has two silent blind spots** — dead-letter queue and ATS-blocks-the-tick.
3. **Real-but-not-urgent ATS duplication** — extract on the *next* source, not now.

---

## Theme 1 — Adding a source has a silent-failure trap (highest impact)

Adding a source touches **four sites**: the new package, `registry.go`, a builder
import, and a builder instantiation block (`internal/sources/builder/build.go`).
Three of those four fail at compile time if you get them wrong. The fourth does not:

- Forget the builder instantiation block (`build.go:65–87` for ATS, `:31–63` for
  filter sources) and the code **still compiles**. The registry classifies the DB
  target correctly, tokens get collected into `boards["x"]`, and then… nothing
  instantiates the source. It is **silently dropped** — no log, no error. A user
  enables the source, and it just never runs.

This is the single fragility in the whole extensibility story. It converges from two
lenses: extensibility calls it a critical silent failure; modularity calls the
matching orchestrator routing a leaky type-check. Both point at the
registry→builder→orchestrator seam.

Two compounding factors:
- **`docs/sources/adding-a-source.md:179–212` is stale** — it documents an env-var
  registration flow (`os.Getenv("MYATS_BOARDS")`) that no longer exists. Sources are
  loaded from the DB (`ListEnabledSourceTargets`). A developer following the doc
  builds something that never wires in.
- **`wis` is hardcoded as an always-on DetailFetcher** (`cmd/worker/main.go:142`) so
  on-demand scrapes work when no wis target is configured. A new HTML source that
  needs the same treatment will silently fail on-demand scrapes, and nothing signals it.

**Direction (cheap, high value):**
- Add a `builder_test.go` that iterates the registry `entries` and asserts every
  source name instantiates through `BuildSources`. This turns the silent runtime drop
  into a red CI check — the highest-leverage single fix in this review.
- Fix `adding-a-source.md` to describe the real DB→registry→builder flow.
- If always-on matters, put an `AlwaysInclude bool` on the registry entry instead of
  hardcoding `wis` in the worker.

---

## Theme 2 — The worker's two blind spots

The worker itself is a reasonable, understandable design: two poll-sleep goroutines
(`worker.Run` drains the job queue with Nack/backoff/dead-letter; `RunScrapeRequests`
handles on-demand). Sleep-based polling over Valkey `ZPOPMIN`/`LPOP` instead of blocking
`BRPOP` is a fine simplicity tradeoff at 6-hour scrape cadence — don't change it.

Two things are worth fixing:

**Dead-letter is a black hole.** URLs that fail 3× land in `jobs:deadletter`
(`queue.go:143–158`) and are **never read, logged, or alerted** — the only references
are the writer and its tests. A source disabled mid-run, or a `CanHandle` that stops
matching a URL format, will quietly accumulate orphaned URLs and job flow stops with
no visible cause. Also: a queued URL that **no** DetailFetcher `CanHandle`s errors →
Nacks → dead-letters, i.e. an orphan is indistinguishable from a transient failure.
*Direction:* a periodic dead-letter drain that logs count + URLs, and a metric to alert
on growth. Small job; converts an invisible failure into a visible one.

**A slow ATS API blocks the whole tick.** ATS sources export **synchronously inside
the cron tick** (`orchestrate.go` atsPath → `BulkExport`). Per-HTTP-call timeout exists
(30s) but there is no per-source ceiling, so one slow/failing board API stalls every
other source in that tick, and there's no circuit breaker on the `/ingest` hop.
*Direction:* wrap each source's tick work in a per-source timeout (and/or run sources
in isolated goroutines so one can't hold up the rest). This is the failure-isolation
gap, not a correctness bug.

Lower priority, noted not urgent:
- `Run` and `RunScrapeRequests` duplicate the poll-sleep-error skeleton with
  undocumented different delays (30s vs 5s empty; no inter-item delay on-demand).
  Fine to leave; document *why* the delays differ.
- Cron tick and on-demand path can scrape the same source concurrently. It's **safe**
  (stateless sources, queue dedups via `ZADD NX`), just wasteful. Accept and add a
  one-line comment, or guard with a per-source lock if it ever matters.

**The API hop itself is justified** — worker and API are separate deployables
(`docker-compose.yml`), so posting to `/ingest` rather than writing the DB directly is
a legitimate boundary, not an accidental network hop.

---

## Theme 3 — ATS duplication is real but do NOT extract yet

The 6 ATS sources (greenhouse, lever, ashby, workable, recruitee, personio) are
~600 lines total and **structurally identical**: same `Iterate` loop-over-boards, same
`fetchBoard` skeleton (build URL → `Get` → `json.Unmarshal` → map fields → `dto.Job`).
The only real variation is the JSON struct shape and the 12–20 line field mapping.
~350 lines are mechanically extractable into a generic `BoardSource[T]` parameterised
by a decode+map func.

Both the modularity and extensibility reviewers independently landed on the same call:
**extract on the next source, not speculatively.** The repetition is stable (not churn),
clear, and debuggable; a generic would hide the one part that actually differs. The
break-even is when you add the 2nd new ATS source in a sprint.

*Direction:* drop a `// ponytail: extractable as BoardSource[T] when the next ATS source lands`
marker at the top of one ATS source so the intent is recorded, and move on.

**What's already good and should not change:**
- `PaginatedBase` (`source.go:63–166`) is a genuinely **deep module** — it hides proxy
  routing (BrightData), client setup, headers, jittered backoff, and offset pagination
  behind a `fetchPage` closure. This is the reuse win already banked.
- LinkedIn's custom pagination (`linkedin.go:113–147`) is **not** duplication to fix —
  its API returns no total count, so `IteratePages` (which needs a total upfront) can't
  apply. It's documented and correct. If a second countless-pagination source appears,
  that's the signal to add an optional `totalCountFn` to `IteratePages`.
- Per-source JSON structs and mapping are correctly package-private. No leakage.

---

## Abstraction leaks to watch (not act on now)

The `Source` interface fits today's sources but will leak on plausible future ones —
worth knowing before you commit to a shape:

- **Pagination:** `IteratePages` assumes numeric offsets. Cursor/GraphQL sources must
  reimplement (LinkedIn already does). Generalise the helper when the 2nd such source arrives.
- **Credentials:** `Config` has no place for API keys/OAuth. First auth'd source will
  invent its own env-var convention. Add a credentials field then, not now.
- **Rate limiting:** each source hardcodes its own 2–7s jitter; not centrally
  configurable or observable. Fine at this scale.

---

## Recommendations roadmap

**Do now (small, high value):**
1. `builder_test.go` asserting every registry entry instantiates — kills the silent-drop trap.
2. Fix `docs/sources/adding-a-source.md` to match the real DB→registry→builder flow.
3. Dead-letter visibility: log/metric on `jobs:deadletter` growth.

**Do soon (failure isolation):**
4. Per-source timeout around tick work so one slow ATS API can't stall the tick.
5. `AlwaysInclude` flag on the registry to replace the hardcoded `wis` DetailFetcher.

**Do later (only when triggered):**
6. Extract `BoardSource[T]` — when the next ATS source lands, not before. Leave a marker.
7. Generalise `IteratePages` for countless pagination — when a 2nd such source appears.
8. `Config` credentials field — when the first authenticated source is added.

**Explicitly leave alone:** `PaginatedBase`, the two-path ATS/HTML orchestration, the
worker/API split, sleep-based polling, and the current per-source ATS code. All are
appropriately simple for the scale.
