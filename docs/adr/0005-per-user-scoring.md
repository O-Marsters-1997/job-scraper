# ADR 0005 — Per-user scoring, gated by interest

`jobs` is a shared catalog with no `user_id`. A score is an assessment of a Job against one User's criteria, like an Application, so it never lives on `jobs`.

- Each User has one `search_config` (criteria, rubric, thresholds), and scores live in `job_scores (job_id, user_id)`. Every scoring step takes the user's config as input, never a constant.
- Cheap per-user exclusion filters from `search_config` run before the next expensive step (a detail fetch or an LLM call), so LLM cost scales with relevant jobs, not board size.
- Suitability scoring runs asynchronously on ingest, whichever path saved the Job. It fans out only to users with an interest signal: a tracked Company (ADR 0016) or an enabled Source Target on that source. A user with no matching interest gets no score.
- A score is redone only when the job content, the user's config or the model changes.
