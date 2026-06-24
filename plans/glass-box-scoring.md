# Plan: Glass-box scoring

> Source PRD: https://github.com/O-Marsters-1997/job-scraper/issues/96

## Technical design decisions

Durable decisions that apply across all phases. Honours **ADR-0005** (phase-agnostic scoring; relevance computed pre-persistence) and **ADR-0006** (per-user scoping of `search_config` and `job_scores`).

### Scoring architecture (current state to build on)
- Suitability scoring runs as a fire-and-forget step in the ingest flow (`Ingest → save (relevance persisted) → score (suitability) → notify`).
- The ingest service is constructed once at startup against a single `SCORING_USER_ID` (`internal/router.go:buildIngestSvc`). `score.IngestScorer.ScoreAndSave` already loads that user's `SearchConfig` — this is the single seam where per-user rubric, cutoff (gate), and model selection are read at score time. v1 scores for one configured user; storage is per-user and multi-tenant-ready (ADR-0006).
- The Claude suitability call (`internal/score/claude.go`) currently hardcodes model `claude-haiku-4-5-20251001` and caps `max_tokens` at 16 to force a bare integer. Capturing reasoning requires raising that cap and moving to a structured (JSON) response.

### Schema (additive, backward-safe)
- **`job_scores`** — add nullable reasoning columns: `reasoning TEXT`, `matched TEXT[]`, `missing TEXT[]` (NULL/empty for rows scored before this feature → drives the "no reasoning captured" empty-state). Add `suitability_skipped BOOLEAN NOT NULL DEFAULT false` (distinguishes "skipped — below cutoff" from "not yet scored"; both have `suitability_score = NULL`).
- **`search_config`** — no schema change; `suitability_rubric`, `relevance_cutoff` (default 0), `notify_threshold` (default 70) already exist. This plan adds the API + UI to edit them.
- **`user_ai_prefs`** — new per-user table (`id`, `user_id UNIQUE FK users`, `suitability_model TEXT NOT NULL DEFAULT '<cheap-default>'`, timestamps). Kept separate from `search_config` to match the dedicated AI settings page and to host future AI prefs. Migrations follow the goose `YYYYMMDDhhmmss_name.sql` convention in `scripts/migrations/`.

### Suitability score states (derived for the UI)
A job's suitability state is one of:
- **scored** — `suitability_score != NULL` and `reasoning != NULL` → show score + reasoning.
- **scored (legacy)** — `suitability_score != NULL` and `reasoning == NULL` → show score, empty-state "no reasoning captured".
- **skipped** — `suitability_score == NULL` and `suitability_skipped == true` → "suitability skipped (below relevance cutoff)".
- **pending** — `suitability_score == NULL` and `suitability_skipped == false` → "not yet scored".

### Key models / interface contracts
- **`SuitabilityScorer`** (`internal/score`): result type extended from a bare `int` to a struct carrying `Score int`, `Matched []string`, `Missing []string`, `Rationale string` (plus existing token usage). The Claude implementation accepts the model id per call.
- **`dto.Job`**: score fields extended to carry `Reasoning *string`, `Matched []string`, `Missing []string`, plus a derived suitability-state indicator (`SuitabilitySkipped bool` is sufficient; UI derives the four states from score + reasoning + skipped). `GET /jobs` is the single read path the list AND detail views consume (detail filters the list client-side), so all reasoning must travel on the `/jobs` response.
- **`dto.SearchConfig`**: already carries `SuitabilityRubric`, `RelevanceCutoff`, `NotifyThreshold`.
- **AI prefs DTO**: `{ SuitabilityModel string }` plus a server-provided list of available Claude models for the picker.

### Routes
- **API** (mirror existing flat per-user resource convention `/source-targets`, `/application-statuses`; all under the `auth.Middleware` group):
  - `GET /scoring-config` → current rubric, cutoff, threshold for the user.
  - `PUT /scoring-config` → update rubric, cutoff, threshold.
  - `GET /ai-prefs` → `{ suitabilityModel, availableModels: [...] }`.
  - `PUT /ai-prefs` → update chosen model.
- **Frontend** (under `frontend/src/routes/_auth/settings/`, mirroring `searches.tsx`/`statuses.tsx`):
  - `settings/scoring` — rubric/cutoff/threshold form.
  - `settings/ai` — model picker.

### Integration points
- Anthropic API via the existing Claude scorer (`ANTHROPIC_API_KEY`); model id now sourced per-user from `user_ai_prefs` at score time.
- Notifier already reads `notify_threshold` from `search_config` (`router.go:setupNotifications`) — making the threshold editable feeds the existing path with no notifier change.

### Testing approach
External-behaviour tests only. DB-touching tests use the existing testcontainers-go pattern (Postgres/Valkey). Extend existing scoring-module tests and `/jobs` handler tests for the new fields; new handler+persistence tests for `/scoring-config`, `/ai-prefs`, and the gate.

---

## Phase 1: Reasoning capture + job-detail display

**User stories**: 1, 2, 3, 5, 6, 7, 8

### What to build

End-to-end forward-only reasoning capture. Add the `reasoning`, `matched`, `missing` columns to `job_scores` (migration). Extend the `SuitabilityScorer` result to the struct form and rewrite the Claude prompt/output to request a JSON object `{ score, matched[], missing[], rationale }`, raising the `max_tokens` cap accordingly. `IngestScorer.ScoreAndSave` persists all four via an extended upsert. Extend the `/jobs` query + `dto.Job` so the reasoning travels on every job row. On the **job detail page**, render the suitability score (currently absent there) plus a reasoning panel (rationale, matched chips, missing chips), with empty-states for legacy-scored (`no reasoning captured`) and not-yet-scored jobs.

### Acceptance criteria

- [ ] Migration adds the three nullable reasoning columns to `job_scores`, backward-safe (existing rows NULL).
- [ ] A job scored above threshold persists a score plus non-empty matched/missing/rationale.
- [ ] `GET /jobs` returns the reasoning fields for scored jobs and nulls/empties for unscored or legacy jobs.
- [ ] The job detail page shows the suitability score and a reasoning panel when present.
- [ ] The detail page shows a clear empty-state for a legacy job (score but no reasoning) distinct from a not-yet-scored job.
- [ ] Scoring-module and `/jobs` handler tests cover the new fields; the prompt change is verified to parse reliably.

---

## Phase 2: Reasoning in the job list (row expander)

**User stories**: 4 (reinforces 7, 8)

### What to build

Add an inline expander to each `/jobs` row that reveals the one-line rationale and compact matched/missing chip lists, consuming the data already delivered by Phase 1's `/jobs` response (predominantly a frontend slice on the existing jobs data table). Expanding must feel instant — no extra fetch. Rows for unscored/legacy jobs show the same honest empty-state copy as the detail panel.

### Acceptance criteria

- [ ] Each scored job row can expand to show rationale + matched/missing chips with no additional network request.
- [ ] Matched/missing render as compact chips/lists, not prose.
- [ ] Unscored and legacy rows show the correct empty-state in the expander.
- [ ] Expander state is per-row and does not disrupt sorting/filtering of the table.

---

## Phase 3: Editable scoring config (Scoring settings page)

**User stories**: 9, 10, 11, 12, 13, 14, 15, 16

### What to build

Expose the already-stored `search_config` fields for editing. Add `GET /scoring-config` and `PUT /scoring-config` (auth-group, per-user) backed by the existing `GetSearchConfig`/`UpsertSearchConfig`. Build a new `settings/scoring` frontend page (mirroring `settings/searches`) with a rubric text area, relevance-cutoff and notify-threshold inputs, plain-language explanations of each, an explicit save confirmation, and a starter-rubric template (pre-filled or one-click insert) plus a new-user empty-state. Edits apply to future scoring only.

### Acceptance criteria

- [ ] `GET /scoring-config` returns the authenticated user's rubric, cutoff, and threshold.
- [ ] `PUT /scoring-config` persists changes and they survive reload; one user's change never affects another's config.
- [ ] The Scoring settings page renders all three fields with explanations and a save confirmation.
- [ ] A new user with no rubric is offered the starter template and sees a meaningful empty-state.
- [ ] Editing the notify threshold feeds the existing notifier path (no regression in notifications).
- [ ] Handler + persistence tests cover read, update, and per-user isolation.

---

## Phase 4: Relevance gate + list toggle

**User stories**: 17, 18, 19, 20

### What to build

Use the relevance cutoff to gate the expensive suitability call. Add `suitability_skipped BOOLEAN` to `job_scores` (migration). In `ScoreAndSave`, after relevance is persisted, compare the job's relevance score to the user's `relevance_cutoff`; if below, skip the Claude call entirely and mark the score row `suitability_skipped = true` (the job remains scraped and stored — never discarded). Surface the distinct "suitability skipped (below relevance cutoff)" state in the list expander and detail panel (not a zero score). Add a job-list toggle to show or hide below-cutoff jobs.

### Acceptance criteria

- [ ] Migration adds `suitability_skipped` (default false), backward-safe.
- [ ] A job with relevance below the user's cutoff is stored, has no suitability score, is marked skipped, and triggers no Claude call.
- [ ] A job at/above the cutoff is scored as before.
- [ ] The UI shows the "skipped — below cutoff" state distinctly from "not yet scored" and from a low score.
- [ ] The job list toggle correctly includes/excludes below-cutoff jobs.
- [ ] A test asserts the suitability call is skipped below cutoff and made at/above it.

---

## Phase 5: Per-user model picker (AI settings page)

**User stories**: 21, 22, 23, 24, 25

### What to build

Make the suitability model per-user, defaulting cheap. Add the `user_ai_prefs` table (migration) with `suitability_model` defaulting to the current cheap model. Thread the model id through the scorer: the Claude implementation takes the model per call, and `ScoreAndSave` reads the user's `user_ai_prefs` (alongside the `SearchConfig` it already loads) and passes the chosen model. Add `GET /ai-prefs` (returning chosen model + curated available Claude model list) and `PUT /ai-prefs`. Build a new `settings/ai` frontend page with the model picker; the default works for users who never open it.

### Acceptance criteria

- [ ] Migration creates `user_ai_prefs` with a cheap default model per user.
- [ ] `GET /ai-prefs` returns the chosen model and the curated list; `PUT /ai-prefs` persists the choice (per-user isolated).
- [ ] The suitability call uses the user's chosen model; with no choice set, it uses the cheap default.
- [ ] A brand-new user (never visited AI settings) still gets working scoring on the default model.
- [ ] The AI settings page renders the picker and a save confirmation.
- [ ] Tests cover the default, an explicit choice being honoured by the scorer, and per-user isolation.
