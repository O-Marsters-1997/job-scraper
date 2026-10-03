# ADR 0021 — Gates, probability-weighted coverage and Bands

Supersedes the scoring formula in [ADR 0004](0004-suitability-scoring.md): "any match in a dimension earns its weight" and the `k=3` prior.

## Context

Suitability resolved each answer to yes, no or unknown at 0.6 and counted a dimension once if any pick matched. In use, 12 of 135 jobs shared the top score of 81. A Staff role met `seniority:senior` and hybrid roles met `work:remote`. One tech match counted the same as seven, and a 60 and a 45 read alike.

## Decision

- **Credit per nice dimension** is `min(1, sum of P(yes) / saturation)` over its picks: 3 for `tech`, 1 elsewhere. **Evidence** is the largest `P(yes) + P(no)` over its picks, so a dimension the judge couldn't read adds little either way. Both use the probabilities. The 0.6 threshold now only labels checklist rows.
- **Score** is `100 * (sum(w * evidence * credit) + 0.5 * prior) / (sum(w * evidence) + avoid cost + prior)` with `prior = 1`. Weights are Go data on `DimensionSpec`: role 3, seniority 3, tech 2, domain 2, work 1, stage 1. An avoid pick costs `2 * P(yes)` in the denominator, so an avoid the job almost certainly lacks still costs a little.
- **Gates cap, they don't zero.** A Gate dimension (seniority, work) caps the score at 44 when every pick the user made in it resolves no and an option they did not pick resolves yes: the job is positively something else. A salary below the floor fires one too, and still pays its avoid cost. The triggering rows get `Effect: "gated"`. A cap keeps the Job visible and sortable, unlike `block`, which still means 0 and hidden. An unknown pick or an unknown unpicked option never fires a Gate, because no answer is not an answer.
- **Gates need the unpicked options answered.** For a Gate dimension with a live pick, `pickedQuestionHashes` covers every live option in the dimension, so the effect asks about them and `FillMissingAnswers` backfills them. That is about 10 extra questions per job. This is also why `multi` dimensions ask every option rather than one enum question: each answer stays atomic and cacheable.
- **Bands** are Great at 80, Good at 65, Fair at 45, Poor below. They are stored in `job_scores.band` by `CompleteAnswerEffect` and `SaveScores`, and are the primary signal in the UI. The number still sorts and checks `notify_threshold`. Cut-offs are hand picks, to be calibrated against Grades.

## Consequences

- Scores spread out and the top ties break up. A single tech match now scores low on its own, since it is a third of the dimension's credit.
- Existing `job_scores` rows have a null Band until the next Recompute or answer effect.
- Gates and weights are Go constants with no per-user override, as in ADR 0004.
