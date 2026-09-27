# ADR 0027 — Shared answers and pick-based Suitability

Suitability came from a single Jev request per (job, user): a free-prose profile, one `noul`
question per hand-written criterion, and an `overall` `score` question over a scale
(ADR 0024). Editing the profile or a criterion marked every score stale
(`score_config_version`), and a rescore asked Jev again for the same job. Picks against a shared
answer bank replace all of it.

- **The bank.** ~160 fixed options (`scoring_options`, seeded by migration) grouped into six
  dimensions (`tech`, `role`, `domain`, `seniority`, `work`, `stage`), each dimension fixed in Go
  code as `pair` (nice/avoid) or `multi` (nice only). Every option is one atomic Jev `choice`
  question (`internal/api/jev`) with `yes`/`no`/`not_stated`, answered once per job and cached in
  `option_answers`, keyed on `(job_id, fingerprint, question_hash, model)`. Answers are shared
  across every interested user; only picks are per user.
- **Suitability is a pure function.** `internal/api/services/suitability.compute` turns a user's
  picks and a job's cached answers into a 0–100 score and one breakdown row per pick: nice picks
  earn their dimension's weight once when any pick in it matches (`met`), avoid picks cost their
  weight only when the job has them, and anything with no known answer counts on neither side. A
  `k=3` prior pulls thin evidence toward 50 so one known match scores 63, not 100.
- **Cost moves to enqueue time, not edit time.** `SaveCanonical` queues one answer effect per job
  in SQL only (`effect_outbox`, now keyed by `(job_id, fingerprint, model)` with no `user_id` or
  `config_version`) if any user is interested. Hard filters and the credential check run when that
  effect is processed, not at enqueue, so ingest no longer needs a Go-side filter pass per user.
  Changing a pick, a blocklist or the notify threshold never calls Jev: `scoringconfig.Update`
  upserts `search_config.preferences` and calls `suitability.Recompute`, which re-scores every job
  the user already has a score for from cached answers alone.
- **Big-bang cutover, no dual-write.** One migration adds `option_answers` and
  `search_config.preferences`, reshapes `effect_outbox` to per-job, adds `job_scores.breakdown`,
  and drops `scoring_questions`, `excluded_seniority`, `job_scores.criteria`, `confidence` and
  `score_config_version`. It deletes every existing `job_scores` row and queues an answer effect
  for every non-closed job with an interested user, so the first tick after deploy rebuilds every
  score under the new formula. There is no rubric to preserve across the cutover, so this mirrors
  ADR 0024's own rollout rather than inventing a migration path for scores that carry no lasting
  value once the judge changes.

This amends ADR 0005: answers are shared per job rather than owned by one user's config, and a
config change no longer marks a user's scores stale — it recomputes them for free instead of
requiring an on-demand rescore. It supersedes ADR 0024's scoring shape (no `overall`, no profile,
`choice` over `noul`), though the `jev` adapter and the OpenRouter credential it bills against
carry over unchanged.

Trade-off: the formula's weights and the 0.6 resolution threshold are Go constants with no per-user
override, and picks can't express anything the bank and a custom question (phase 4) don't cover —
that's deliberate, since atomic shared questions are what make answers cacheable and picks free to
change.
