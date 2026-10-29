# ADR 0039 — Five-level Picks, never Gates and Custom Options

Amends [ADR 0021](0021-gates-weighted-coverage-and-bands.md) (Gate dimensions, block) and the shared-bank assumption of [ADR 0004](0004-suitability-scoring.md).

## Context

Scoring v2 settings show every Option as a 1–5 scale. Today's stances don't line up with it. `block` exists only for domain. Work and employment Gates fire on an option the User did *not* pick, so leaving an option unrated is not neutral. Some preferences are too specific to hand-curate in a shared list, for example "building agents and LLM product features" as opposed to "ML, evals and RAG infrastructure".

## Decision

- **Five levels, one stance each.** 1 `never`, 2 `avoid`, 3 no Pick, 4 `ok`, 5 `nice`. Avoid, ok and nice keep their ADR 0021 maths. Anything unrated is a 3 and has no effect.
- **Never is a per-option Gate.** A never Pick that resolves yes caps the score at 44 and keeps the Job visible, on any Dimension. `block` (score 0, hidden) retires, and existing block Picks migrate to never. A misread answer stays recoverable through a Correction instead of hiding the Job.
- **The dimension Gate retires.** "Every pick no and an unpicked option yes" would make a 3 act like a 1. Gates now come only from never Picks, the Ladder's spread and the salary floor. Unpicked options in work and employment are no longer asked, which saves about 10 Jev questions per Job.
- **Tech saturation is per User.** "A Job needs N" replaces the constant 3. There is no separate core flag: 5 against 4 already weights core against nice-to-have.
- **Custom Options.** `scoring_options` gains a nullable owner. Owned Options join an existing Dimension, so they inherit its weight, its saturation and its `fit` weight. Only the owner sees them, and an admin can promote one by clearing the owner. A User drafts one from a phrase: a model writes the question and suggests a Dimension, `scoring.Ask` previews it on about 5 recent Jobs, and the User confirms. A User can have at most 10 live. An edit retires the Option and replaces it, because Answers are keyed on the question text.
- Conditional preferences ("Backend only in my languages") and the AI split are Custom Options, not a new conditional-Pick mechanism.

## Considered

- A single `custom` Dimension: unrelated Options would compete for one saturation slot, and `fit` would learn one weight for a grab-bag.
- Extending block to every Dimension: one wrong Answer would silently hide a Job.
- A general "counts only alongside" Pick: a new Pick shape and new breakdown rows for what one Custom Option already expresses.

## Consequences

- `loadBank` becomes per User: shared Options plus the User's own.
- The 10-Option cap bounds the added Jev cost at about what the retired dimension Gate used to ask.
- Question quality now rests partly on Users. The draft and preview step is the guard (#1540).
- Learning Custom Options from Dismiss reasons is deferred.
