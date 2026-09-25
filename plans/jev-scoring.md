# Plan: Jev suitability scoring

> Source: grilling session 2026-09-25. Jev docs: https://openrouter.ai/docs/guides/community/jev-tutorial

Replace Claude as the suitability judge with Jev (`typesafe/jev-1.13`, TypeSafe) via OpenRouter's
decisions endpoint. Jev answers typed questions about a `state` object and returns probabilities,
with no prose. Billing is input tokens only.

## Technical design decisions

Durable decisions that apply across all phases. Honours **ADR-0005** (per-user scoring, cheap
filters before expensive steps, model in the staleness key) and **ADR-0022** (scoring lives in
`cmd/api` behind `/ingest`).

### Jev only, no prose
- Jev is the only scorer. No rationale is generated anywhere, on demand or otherwise.
- `ClaudeScorer`, the `jobreasoning` endpoint and its UI, `anthropic-sdk-go`, `ingest.ProviderForModel`
  and the aiprefs model allowlist are deleted at cutover (Phase 3).

### Wire contract
- `POST https://openrouter.ai/api/alpha/decisions`, raw `net/http` (no Go SDK). The endpoint is
  `alpha`: keep the request/response structs private to `internal/score` so a shape change is one file.
- Request `model` is the Go constant `typesafe/jev-1.13`. The response's dated `model`
  (e.g. `typesafe/jev-1.13-20260917`) is stored in `job_scores.score_model`. A version bump is a
  one-line change and re-queues every score through the existing model key.
- One job per request. All questions in the request are answered in parallel by Jev.

### Questions sent per job
Built from the user's `scoring_questions`:
- One `noul` question per criterion: `instructions` plus `criteria: {true, false}`.
- One `score` question named `overall`: fixed instructions ("How well does this job fit what the
  candidate is looking for?") with `criteria` = the user's ordered scale.

### State sent per job
```json
{
  "job": { "title", "company", "location", "work_arrangement", "salary_raw", "description" },
  "candidate": { "profile": "<~100-word what-I'm-looking-for>" }
}
```
- `description` is HTML-stripped and truncated on a rune boundary (fixes the current byte cut).
- The job text is a JSON string value in `state`, never an instruction, which closes the
  prompt-injection hole in the current user-turn prompt.
- `salary_raw` and `location` reach the judge for the first time (ADR-0007 intent).

### Score formula
- `overall = answers.overall.score / (len(scale) - 1)`, in 0..1.
- `suitability = round(100 × overall × min(P(yes) over required criteria))`, where the min over an
  empty set is 1. A clear miss on a required criterion sinks the score; uncertainty only dampens it.
- Matched/missing are derived at P(yes) ≥ 0.5 / < 0.5 in the DTO, not stored.

### Schema
- **`search_config`**: replace `suitability_rubric text` with `scoring_questions jsonb`:
  ```json
  {
    "profile": "string",
    "criteria": [{ "key": "go_backend", "instructions": "...", "true": "...", "false": "...", "required": false }],
    "scale": ["Not relevant", "Weak", "Possible", "Strong", "Apply today"]
  }
  ```
  Validated in `services/scoringconfig`: unique slug keys, 2..N scale levels, non-empty profile or
  at least one criterion. `updated_at` still drives staleness and rescoring.
- **`job_scores`**: add `criteria jsonb` (`{key: P(yes)}`), `confidence real` (from `overall`),
  `cost numeric` (from `usage.cost`). At cutover, drop `reasoning`, `matched`, `missing`,
  `relevance_score`, `suitability_skipped`.
- **`user_ai_prefs`**: drop `suitability_model` and `reasoning_model` at cutover. Drop the table if
  nothing else lives in it.
- Replace the hardcoded Haiku `COALESCE` defaults in `effect_outbox.sql` (about 6 places) with the
  model passed in from Go.

### Credentials
- Per-user BYOK: `user_ai_credentials` with `provider = 'openrouter'`, through the existing
  `credstore`. The outbox fetches `cs.Get(user, "openrouter")`; the hardcoded `"anthropic"` in
  `cmd/api/main.go` goes away.
- Effects are only queued for users with an `openrouter` credential.

### Key interfaces
- `score.SuitabilityScorer` stays as the seam. `SuitabilityResult` becomes
  `{Score int, Criteria map[string]float64, Confidence float64, Model string, Cost float64}`.
- `JevScorer` builds the questions and state from `(job, scoring_questions)`, calls the endpoint,
  and applies the score formula. The formula is a pure function with a table test.

---

## Phase 1: Scoring pipeline fixes (provider-independent)

### What to build
Fixes that pay off before Jev lands and that the backlog after cutover depends on.
- **Reject filters at enqueue**: before `QueueScoringEffects` fans out, run `score.Reject` per user
  and skip users whose filters exclude the job. The fan-out loads user configs in Go instead of a
  pure `INSERT … SELECT`.
- **Drain per tick with a small pool**: each tick claims effects until the queue is empty, with
  `errgroup.SetLimit(4)`. The `SKIP LOCKED` claim and lease are unchanged.
- **Non-retryable failures**: 401/402 (bad key, no credit) fail the effect at once. A 429 fails it
  with a `Retry-After` backoff instead of the exponential one.
- **Don't queue effects for users with no credential** for the scorer's provider.
- **`.env.example`**: add `AI_CREDENTIAL_ENC_KEY`, remove the unused `ANTHROPIC_API_KEY`.

### Acceptance criteria
- [ ] A board/ATS job matching a user's excluded company, keyword, seniority or location gets no effect for that user.
- [ ] A tick with N pending effects processes all of them with at most 4 calls in flight.
- [ ] A 401/402 marks the effect `failed` after one attempt; a 429 schedules the retry at `Retry-After`.
- [ ] A user with no credential gets no effects queued.
- [ ] Tests use `providers.Mock*` and a fake scorer; no HTTP to a real provider.

---

## Phase 2: JevScorer, config editor and shadow compare

### What to build
- `JevScorer` in `internal/score/jev.go` (raw HTTP, private wire structs, score formula).
- Migration: add `search_config.scoring_questions` and seed it from `suitability_rubric`
  (profile = old rubric text, criteria = `[]`, the default 5-level scale). Add
  `job_scores.criteria`, `confidence` and `cost`. Mirror in `internal/data/sqlc/schema.sql` and run
  `just generate`.
- `scoringconfig` service validates and saves `scoring_questions`; the DTO carries it.
- Settings UI: a profile textarea, a criteria list editor (instructions, true/false wording,
  required toggle) and a scale editor. Replaces the rubric textarea.
- AI credentials UI accepts an OpenRouter key.
- Throwaway CLI `just cli jev-compare`: scores the last ~200 already-scored jobs with Jev for the
  current user and prints the Spearman rank correlation against the stored Claude scores plus the
  20 biggest disagreements (title, Claude, Jev, per-criterion P). It is not wired into the outbox.

### Acceptance criteria
- [ ] `JevScorer` against an `httptest` server: the request carries one `noul` per criterion and one `overall` score question; the state carries the job fields and profile.
- [ ] Score formula table test: no criteria; required criterion at 0.1; scale of 2 and of 5 levels.
- [ ] Description truncation never splits a rune.
- [ ] The migration seeds `scoring_questions` from the existing rubric; `GET /scoring-config` returns it.
- [ ] `PUT /scoring-config` rejects duplicate keys, a scale with fewer than 2 levels, and an empty profile with no criteria.
- [ ] `jev-compare` runs against your live data and you have reviewed its disagreements before Phase 3.

---

## Phase 3: Cutover and Claude removal

### What to build
- Wire `JevScorer` into the outbox with `provider = "openrouter"`, persisting `criteria`,
  `confidence`, `cost` and the dated model. Existing scores go stale via the model key and rescore.
- Delete `ClaudeScorer`, the `jobreasoning` service, route and UI, the `anthropic-sdk-go` dep,
  `ingest.ProviderForModel`, the aiprefs model allowlist and model picker, and the `jev-compare` CLI.
- Migration drops `search_config.suitability_rubric`, `job_scores.reasoning/matched/missing/relevance_score/suitability_skipped`
  and the `user_ai_prefs` model columns.
- Job list and detail views show per-criterion probability bars and flag low-confidence scores.
  Matched/missing chips are derived at 0.5.
- ADR: "Jev as the suitability judge". Update the CONTEXT.md **Suitability** entry (it currently
  says "LLM (Claude Haiku)").

### Acceptance criteria
- [ ] A newly ingested job for a user with an OpenRouter key is scored by Jev end to end; `job_scores` holds the score, criteria, confidence, cost and the dated model.
- [ ] Every pre-existing score is re-queued and rescored after deploy.
- [ ] Alert emails still fire at `score ≥ notify_threshold` on first discovery.
- [ ] `go.mod` no longer references `anthropic-sdk-go`; `rg -i 'claude-|anthropic' internal cmd` finds nothing.
- [ ] `sqlc generate && git diff --exit-code` is clean; `just lint` and `just test` pass.
- [ ] The ADR is written and CONTEXT.md is updated.
