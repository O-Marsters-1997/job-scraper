# ADR 0024 — Jev as the suitability judge

Claude Haiku scored suitability by asking a free-text-JSON-mimicking chat model for a
0-100 score, matched/missing lists and a rationale, then parsing whatever came back. Jev
(`typesafe/jev-1.13`, TypeSafe, via OpenRouter's `/api/alpha/decisions` endpoint) replaces it as
the only suitability judge. Jev answers typed yes/no and scaled questions against a `state`
object and returns calibrated probabilities — no prose, no JSON-parsing of a chat response.

- `JevScorer` (`internal/score/jev.go`) builds one `noul` (yes/no) question per criterion from the
  user's `scoring_questions` plus one `score` question, `overall`, over the user's ordered scale.
  `state` carries structured job fields (title, company, location, work arrangement, salary,
  HTML-stripped description truncated on a rune boundary) and the candidate's profile as JSON
  values, never as instructions — closing the prompt-injection hole the old free-text user-turn
  prompt had.
- Suitability is `100 × overall × min(P(yes) over required criteria)`, where the min over no
  required criteria is 1: a clear miss on a required criterion sinks the score, uncertainty only
  dampens it. `job_scores` gains `criteria` (per-key `P(yes)`), `confidence` (the `overall`
  answer) and `cost`; matched/missing become a ≥0.5 read of `criteria` in the DTO rather than a
  second thing to store.
- The endpoint is `alpha`, so its request/response structs stay private to `internal/score` — a
  shape change is a one-file diff. The request's `model` is the Go constant `typesafe/jev-1.13`;
  the response's dated snapshot (e.g. `typesafe/jev-1.13-20260917`) is what lands in
  `job_scores.score_model`, so staleness checks compare against the constant's prefix instead of
  exact equality (a version bump changes the prefix and re-queues every score, honouring ADR
  0005's "model changes" trigger without a second model column).
- Credentials move to `user_ai_credentials` under provider `"openrouter"`; effects are only queued
  for users who have one.

There is no shadow-compare or dual-path rollout: there was no score data worth preserving, so
`ClaudeScorer`, the `jobreasoning` endpoint and its UI, `anthropic-sdk-go`, `ingest.ProviderForModel`
and the aiprefs model allowlist were deleted in the same change that wired Jev in, and one
migration both adds the new `job_scores` columns and drops `search_config.suitability_rubric`,
the old `job_scores` reasoning/matched/missing/relevance/skip columns, and `user_ai_prefs`
(dropped entirely — nothing else lived in it).

Trade-off: an `alpha` endpoint can change shape without notice, and Jev gives no rationale text at
all, so a rescored job has a number and per-criterion probabilities but no explanation a user can
read.
