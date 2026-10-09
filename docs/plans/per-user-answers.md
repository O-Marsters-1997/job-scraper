# Plan: Per-User Answers

> Source: `/grill-me` session on spike #407; [ADR 0023](../adr/0023-answers-and-scores-are-per-user.md)
> Label: project:per-user-answers

## Technical design decisions

- **Problem**: `answerMissing` asks every surviving User's missing questions on the first keyed User's
  OpenRouter key, Answers are shared across Users, and the full cost is written to every User's score row.
  Scheduled discovery (ADR 0022) makes this background spend continuous.
- **Guarantee**: a User's key only ever pays for, and carries, that User's Picks. No Answer, Correction,
  cost or breakdown is visible to or reused by another User. Company-profile-derived Answers stay shared
  because they're not Jev output.
- **Schema** (scoring context, one migration, mirrored into `schema.sql`):
  - `option_answers` gains `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`. The PK becomes
    `(user_id, job_id, fingerprint, question_hash, model)`.
  - Backfill: insert one copy of each existing row per User with a `job_scores` row for that `job_id`
    whose `score_fingerprint` matches. Then delete the ownerless rows and set NOT NULL.
  - `effect_outbox` gains `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`. The pending
    unique index becomes `(job_id, user_id, fingerprint, model) WHERE status IN ('pending','running')`.
    Existing pending rows are expanded to one per interested User, using the same query as below.
- **Key models**: `dto.AnswerEffect` gains `UserID`. `dto.JobScore.Cost` is that User's own spend.
- **Module boundaries** (all in `internal/services/scoring`):
  - The tx-scoped ports `JobsChanged(ctx, tx, jobIDs, firstDiscovery)` and `CompanyTracked(ctx, tx, userID,
    companyID)` keep their signatures and insert one outbox row per interested User. Interest comes from
    the query behind `ListInterestedConfigs`, made set-based over job IDs.
  - The effect processor handles one (Job, User). It loads that User's config, applies Relevance and the age
    cutoff, and stops if a score already exists at this fingerprint. With no key it completes with no score
    row. Otherwise it asks Jev only that User's missing Picks on their key, then writes their Answers, score
    and cost in one transaction. Notification logic is unchanged and per User.
  - `Ask(ctx, userID, jobID, questions)` reads and writes only `userID`'s Answers.
  - `FillMissingAnswers(ctx, userID)` queues effects only for `userID`.
  - Store reads of Answers (`ListAnswers`, `ListScoringAnswersForUser`) take `userID`.
- **Failure**: retries and terminal failures are per outbox row, so one User's dead or rate-limited key
  never blocks another's.
- **Frontend**: a Job the User is interested in but can't score (no key) shows "Connect an OpenRouter key
  to score jobs" in place of the score. The API exposes this as the existing null `SuitabilityScore` plus
  `ScoringStatus`.
- **Glossary**: done (`CONTEXT.md` **Answer**).
- **Tests**: scoring service tests with `scoringtest.FakeStore` (two Users with overlapping Picks: each
  pays for and stores only their own; a keyless User gets no row; one key failing doesn't block the
  other), plus a pgtest for the migration backfill and the per-User outbox enqueue.

---

## Phase 1: Per-(Job, User) answer effects

**User stories**: background scoring bills each User only for their own Picks, and nothing crosses between
Users.

### What to build

The migration with its backfill, `user_id` through the outbox, ports, processor and store, and per-User
cost on score rows and on the `score call` log line.

### Acceptance criteria

- [ ] For two Users with overlapping Picks on one new Job, each User's key gets one Jev call containing
      only their Picks, and each stores their own Answers.
- [ ] A User with no key gets no `job_scores` row, and the other User is still scored.
- [ ] A rate-limited key fails only its own outbox row.
- [ ] `job_scores.cost` and the `score call` log equal that User's own spend.
- [ ] The migration copies existing Answers only to Users scored at the same fingerprint (pgtest).

---

## Phase 2: User-initiated paths are per User

**User stories**: changing Picks and asking questions spend only the caller's key and read only the
caller's Answers.

### What to build

`Ask` and `FillMissingAnswers` scoped to the calling User, and the Pick-change backfill enqueueing that
User's effects only.

### Acceptance criteria

- [ ] After User A adds a Pick, only A's key is charged for the backfill, and B's scores are untouched.
- [ ] `Ask` by A never returns an Answer that B paid for.

---

## Phase 3: Unscored state in the UI

**User stories**: a User without an OpenRouter key understands why their Jobs have no score.

### What to build

In the jobs list and job detail, when the User has no key, unscored Jobs show a prompt linking to Settings → AI
in place of the score. Read `DESIGN.md` first.

### Acceptance criteria

- [ ] With no credential, unscored Jobs show the connect-a-key prompt.
- [ ] Once a key is added and effects run, scores replace the prompt.
