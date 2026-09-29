# Plan: Settings → Searches redesign

> Source: grilling session on 2026-09-29 over the `prototype/searches-ux` prototype
> (`frontend/src/routes/_auth/settings/-searches-prototype/`) and
> `.claude/handoffs/job-scraper__searches-ux-grilling.md`.

Before phase 1, commit the prototype to a throwaway branch (`prototype/searches-ux-reference`) as the
visual reference. Nothing from it merges as-is; each phase rewrites the parts it needs.

## Technical design decisions

- **Routes (frontend)**: `/settings/searches` is the single place to manage what gets scraped. Search
  params (`validateSearch`): `tab` (`"boards"` default | `"ats"`), `q`, filter chips, `sort`, `dir`, `page`.
  `/companies` becomes a browse-only directory (its add form goes; the detail page keeps its tracking switch).
  `/jobs` gains `company=<id>` and `scored=1` in its URL state.
- **Two tabs, two models**:
  - *Job board searches* are discovery `source_targets` (LinkedIn, Indeed, WIS, RemoteOK, Remotive).
  - *Company boards* are `tracked_companies` rows, one per tracked Company, showing its `company_boards`.
    ATS sources stop being valid `source_targets` (phase 3).
- **Search storage** (unchanged schema): a pasted URL is parsed server-side into `value` + named `filters`
  for `filter`-kind sources (LinkedIn, WIS). Indeed stays `kind: url`, stored as a normalised URL (tracking
  params and `start=` stripped). Params the scraper can't use are dropped and reported back, never stored.
  Dedupe stays `UNIQUE(user_id, source, value, filters)`.
- **Schema**: no new tables. Phase 3 adds a data migration (ATS `source_targets` → `tracked_companies`,
  then delete them). Phase 5 adds an index to support the scored count if `EXPLAIN` shows a need
  (`job_scores (user_id, job_id)` is already covered by the `UNIQUE (job_id, user_id)`; check).
- **Module boundaries**:
  - `internal/detect` (shared kernel, ADR 0003) owns URL recognition. New: `ParseSearchURL(raw) (SearchURL, bool)`
    and `BuildSearchURL(source, value, filters) string` for LinkedIn/Indeed/WIS. Pure, no I/O. It is a deep
    module and gets isolated table tests with real pasted URLs (LinkedIn `/jobs/search-results/`, tracking
    params, later-page URLs, unknown params).
  - `internal/sourcespec` owns filter vocabularies. `FilterField` gains `Options []FilterOption{Value, Label}`
    so the form's selects come from `GET /sources`, and the LinkedIn/WIS adapters validate against the same
    table instead of their own copies. `detect` maps URL params (`f_TPR`, `f_WT`, `geoId`…) onto these names.
  - `jobsearch` owns everything here: source targets, companies, tracking, and the list queries. It
    read-joins scoring's `job_scores` for the relevant count (allowed by ADR 0011).
  - The worker is untouched except the adapters reading filter vocabularies from `sourcespec`.
- **API contracts**:
  - `GET /sources/resolve?url=` → `dto.ResolvedURL`:
    `{kind: "search" | "ats", source, value, filters: {name: value}, dropped: [param], url}`. `422` with
    a specific message for recognised-but-unsupported hosts (RemoteOK, Remotive) vs. unrecognised URLs.
  - `dto.SourceTarget` gains `URL string` (canonical, from `BuildSearchURL`); `GET /source-targets` moves
    from a bare `GetAll` to a service method that fills it.
  - `POST /source-targets` normalises Indeed URLs and, from phase 3, rejects ATS sources.
    A duplicate returns `409`.
  - `GET /companies/tracked` → `[]dto.TrackedCompany`:
    `{id, name, slug, enabled, check_interval_minutes, boards: [{id, source, board_token, status, url}],
    open_jobs, relevant_jobs, last_checked_at}`.
  - `DELETE /companies/{id}/tracking` untracks (deletes the `tracked_companies` row). Pause stays
    `PUT /companies/{id}/tracking {enabled}`. Confirm board stays `POST /companies/{id}/boards {url, confirm: true}`.
  - `GET /jobs` (`dto.JobsQuery`) gains `scored` (`"1"`: only jobs with a `job_scores` row for the caller).
- **Delete with undo (frontend)**: the row hides immediately and a toast offers Undo for ~5s. The DELETE is
  sent when the toast expires, or at once on route unmount / `pagehide`. No backend support.
- **Paging**: both lists are fetched whole and paged, filtered and sorted in the browser (25 per page for
  companies, 10 for searches). Revisit server paging past ~2k tracked companies.
- **Relevance**: "relevant" means passed `filter.Reject` against the user's Search Config, which is exactly
  "has a `job_scores` row". CONTEXT.md is corrected to match (no numeric cutoff).

---

## Phase 1: Board searches tab on real data

**User stories**: see and manage my job board searches; build a search from fields; run, pause and delete
one; keep my place in the table across refresh and back.

### What to build

Replace the prototype route with the real page: segmented tabs, and the Job board searches table
(search, source chips, sortable headers, pager) over `GET /source-targets`. "Build from fields" opens the
search form with a source picker; its selects render from `GET /sources`, which now carries filter
`Options` from `sourcespec`. LinkedIn/WIS adapters read the same vocabularies. Create, toggle (PATCH),
Run now (`POST /{id}/scrape`) and delete go through the existing endpoints with real mutations and query
invalidation. Delete uses the undo toast. All table state lives in the route's search params.
The Company boards tab shows a link to `/companies` until phase 4. Fix `SortableTableHead` so headers keep
their uppercase styling (remove the prototype's local workaround). Delete `useProtoSearches` and the fake padding.

### Acceptance criteria

- [ ] Creating a search from fields queues its first run and the row shows the run status
- [ ] Toggle, Run now (409 surfaces as "already running") and delete work against the API
- [ ] Undo restores a row with no request sent; leaving the page within the undo window still deletes
- [ ] `tab`, `q`, filters, sort and page survive refresh and back/forward
- [ ] Filter selects come from `GET /sources`; LinkedIn adapter tests pass using the `sourcespec` vocabulary
- [ ] Checked at 390px and desktop widths
- [ ] Old settings/searches page code removed

---

## Phase 2: Paste a search URL

**User stories**: paste a LinkedIn/Indeed/WIS search URL and get an editable search; see which URL
params won't be used; open a saved search on the board.

### What to build

Add `ParseSearchURL` and `BuildSearchURL` to `internal/detect`, covering the prototype's `boardUrl.ts`
rules (tracking params dropped, `/jobs/search-results/` → `/jobs/search/`, `start=`/`p=` dropped). Widen
`GET /sources/resolve` to return `dto.ResolvedURL` for both searches and ATS boards. Create normalises Indeed
URLs through the same code. `GET /source-targets` returns each target's canonical `URL`, shown as an
"Open on <board>" link on the row. The paste box calls resolve (debounced), switches to the matching tab,
opens the form pre-filled, lists dropped params, and blocks duplicates using the list it already has
(server 409 as backstop). After the paste the form edits fields only. RemoteOK/Remotive URLs get
"not supported yet — use Build from fields". Delete `boardUrl.ts`. Write ADR 0015
"Board search URLs parse into named filters".

### Acceptance criteria

- [ ] Table tests for `ParseSearchURL` using real pasted URLs, including the user's LinkedIn URL
- [ ] `BuildSearchURL(ParseSearchURL(u))` round-trips for every supported filter
- [ ] Two Indeed URLs differing only in tracking params dedupe to one target
- [ ] Unsupported and unrecognised URLs show distinct messages
- [ ] ADR 0015 written

---

## Phase 3: Retire ATS source targets

**User stories**: tracking a company is one thing, not two; no behaviour change visible to the user.

### What to build

A migration upserts a `tracked_companies` row (enabled and interval carried over) for every ATS-source
`source_targets` row, via the matching `companies` row, then deletes the ATS targets. `POST /source-targets`
rejects ATS sources with a pointer to Company boards. `SetCompanyTracking` stops calling
`UpsertSourceTargetForCompany` (delete the query). `ListCompaniesForUser` takes last check from
`board_poll_state` instead of the lateral join on `source_targets`. Remove the ATS branches from
`sourcetargets.Service` (`Scrape`, `enqueueRun`, `publishRun`) and the `source_targets` ATS clause
(`st.value = j.company_slug`) from scoring's `ListInterestedConfigs` and `QueueAnswerEffect`. Mirror the
migration in `schema.sql` where needed; regenerate sqlc. Amend ADR 0007: ATS sources are not valid Source
Targets, and tracking is managed from Settings → Searches.

### Acceptance criteria

- [ ] Migration test: an ATS target becomes a tracked company with the same enabled flag and interval,
      and no ATS `source_targets` rows remain
- [ ] A job on a tracked company's board is still queued for scoring for that user
- [ ] `/companies` and its detail page still show tracking and last check
- [ ] `sqlc generate && git diff --exit-code` clean; ADR 0007 amended

---

## Phase 4: Company boards tab

**User stories**: see all the companies I track in one table; paste an ATS URL to track a company; pause
or untrack one; confirm an unverified board.

### What to build

Add `ListTrackedCompaniesForUser` (tracked rows only, with boards, `open_jobs` = open jobs by
`company_id`, last check) behind `GET /companies/tracked`, and `DELETE /companies/{id}/tracking`.
The Company boards tab renders `dto.TrackedCompany` rows: ATS badges link out to the board, the name
links to `/companies/$id`, and the open count links to `/jobs?company=<id>`. Pasting an ATS URL in the
paste box shows "Track company", which calls `POST /companies {url, track: true}`. The toggle pauses via
`PUT /companies/{id}/tracking`; delete untracks with the undo toast. A board with status `candidate` shows
an "Unverified — not polling yet" chip and a "Confirm board" action (`POST /companies/{id}/boards
{url, confirm: true}`). Remove the add-by-URL form from `/companies`.

### Acceptance criteria

- [ ] Pasting a Greenhouse URL tracks the company and the row appears with an Unverified chip
- [ ] Confirm board queues verification; the chip clears once the board is verified
- [ ] Pause keeps the row and interval; untrack removes it; undo works as in phase 1
- [ ] Store test for `ListTrackedCompaniesForUser` (only this user's tracked companies, correct open count)
- [ ] 232-row list pages, searches and sorts client-side without lag; checked at 390px

---

## Phase 5: Relevant roles

**User stories**: see how many open roles at each tracked company are relevant to me, and click through
to exactly those jobs.

### What to build

Add `relevant_jobs` to `ListTrackedCompaniesForUser`: open jobs for the company with a `job_scores` row
for the user. Add `scored` to `dto.JobsQuery` and `PageJobs` (an `EXISTS job_scores` condition for the
caller), and `company`/`scored` to the `/jobs` URL state (`lib/jobFilters.ts` and the jobs API call).
The row shows "**N** of M open" linking to `/jobs?company=<id>&scored=1`. Correct CONTEXT.md: Relevance is
pass/fail against Search Config exclusions, checked before Suitability scoring, with no cutoff; remove the
relevance cutoff from Search Config.

### Acceptance criteria

- [ ] Store test: N counts only open, scored-for-this-user jobs of that company
- [ ] `/jobs?company=<id>&scored=1` lists exactly N jobs
- [ ] `EXPLAIN` of the tracked list is acceptable at a few hundred companies
- [ ] CONTEXT.md Relevance and Search Config entries updated
