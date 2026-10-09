# ADR 0023 — Answers and scores are per User

Reverses the shared Answer in `CONTEXT.md`. Prompted by spike #407, because scheduled runs score in the
background with no User at the keyboard.

`answerMissing` (`internal/services/scoring/service.go`) takes the first surviving User with an OpenRouter
key and asks every missing question on that one key, the other Users' Picks included. The Pick-change
backfill (`FillMissingAnswers`) goes through the same path. So one User pays for another's Picks, which
User pays is an accident of row order, and B's questions show up in A's OpenRouter activity. `score.Cost`
and the `score call` log line then charge the whole cost to every User.

## Decision

- **A User's key only ever pays for, and carries, that User's Picks.** Nothing scoring-related crosses
  between Users: Answers, Corrections, costs and breakdowns.
- **`option_answers` gains `user_id`.** An Answer is keyed on User, Job, content fingerprint, question
  hash and model.
- **The answer effect is per (Job, User).** For each interested User who passes Relevance and the age
  cutoff: if they already have a score at this fingerprint, skip. If they have no OpenRouter key, skip and
  write no score row. Otherwise ask Jev their missing Picks on their key, and store their Answers, score
  and cost. One User's failing key never blocks another's.
- **The Job stays shared.** Ingest upserts it, and duplicates collapse to one canonical Job as before.
- **Company-profile-derived Answers stay shared.** They come from Company facts, not from Jev, so they
  reveal nothing about another User.

## Rejected

- **One shared call for the union of Picks:** cheapest in total, but it bills and exposes one User's Picks
  on another's key.
- **Per-User keys over a shared Answer cache:** a cache hit reveals that another User asked that question
  about this Job.
- **Billing the User whose search found the Job:** a Job found by two searches or an ATS Board has no
  single owner, and Pick-change backfill has no search at all.
- **An operator key for background scoring:** the operator ends up paying for every User's Picks.

## Consequences

- An overlapping Pick is paid for once per User, and the description is sent once per User. Each User's
  bill equals what they would pay as the only User.
- A User without a key sees their matched Jobs unscored rather than scored on unknowns.
- Changing your Picks backfills on your own key only.
- Existing shared `option_answers` rows have no owner. The migration copies each one to every User who
  has a `job_scores` row for that Job at the same fingerprint, then drops the ownerless rows. Nobody pays
  again, and nobody gains an Answer that their own score hadn't already used.
