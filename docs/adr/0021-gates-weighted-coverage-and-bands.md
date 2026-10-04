# ADR 0021 — Gates, probability-weighted coverage and Bands

Supersedes the scoring formula in [ADR 0004](0004-suitability-scoring.md): "any match in a dimension earns its weight" and the `k=3` prior.

## Context

Suitability resolved each answer to yes, no or unknown at 0.6 and counted a dimension once if any pick matched. In use, 12 of 135 jobs shared the top score of 81. A Staff role met `seniority:senior` and hybrid roles met `work:remote`. One tech match counted the same as seven, and a 60 and a 45 read alike.

## Decision

- **Credit per nice dimension** is `min(1, sum of P(yes) / saturation)` over its picks: 3 for `tech`, 1 elsewhere. **Evidence** is the largest `P(yes) + P(no)` over its picks, so a dimension the judge couldn't read adds little either way. Both use the probabilities. The 0.6 threshold now only labels checklist rows.
- **Score** is `100 * (sum(w * evidence * credit) + 0.5 * prior) / (sum(w * evidence) + avoid cost + prior)` with `prior = 1`. Weights are Go data on `DimensionSpec`: role 3, seniority 3, tech 2, domain 2, work 1, stage 1. An avoid pick costs `2 * P(yes)` in the denominator, so an avoid the job almost certainly lacks still costs a little. A picked dimension's missing evidence now counts as a coin flip (see the amendment below).
- **Gates cap, they don't zero.** A Gate dimension (seniority, work) caps the score at 44 when every pick the user made in it resolves no and an option they did not pick resolves yes: the job is positively something else. A salary below the floor fires one too, and still pays its avoid cost. The triggering rows get `Effect: "gated"`. A cap keeps the Job visible and sortable, unlike `block`, which still means 0 and hidden. An unknown pick or an unknown unpicked option never fires a Gate, because no answer is not an answer.
- **Gates need the unpicked options answered.** For a Gate dimension with a live pick, `pickedQuestionHashes` covers every live option in the dimension, so the effect asks about them and `FillMissingAnswers` backfills them. That is about 10 extra questions per job. This is also why `multi` dimensions ask every option rather than one enum question: each answer stays atomic and cacheable.
- **Bands** are Great at 80, Good at 65, Fair at 45, Poor below. They are stored in `job_scores.band` by `CompleteAnswerEffect` and `SaveScores`, and are the primary signal in the UI. The number still sorts and checks `notify_threshold`. Cut-offs are hand picks, to be calibrated against Grades.

## Consequences

- Scores spread out and the top ties break up. A single tech match now scores low on its own, since it is a third of the dimension's credit.
- Existing `job_scores` rows have a null Band until the next Recompute or answer effect.
- Gates and weights are Go constants with no per-user override, as in ADR 0004.

## Amendment: favourite Companies (β)

A favourite Company lifts the pre-gate score in log-odds: `p = clamp(score / 100, 0.01, 0.99)`, `score' = round(100 * sigmoid(logit(p) + β))`, never below the unlifted score. It applies before the Gate cap and the block, so a favourite never rescues a gated or blocked Job, and it adds a `company:favourite` row with `Effect: "favourite"`. The shift is near zero at either end and largest across Fair and Good. `β` is `favouriteBeta` in `compute.go`.

β is 0.4: about +10 at a score of 50 and +5 at 85. Replay against 46 Grades (28 positive, 18 negative) with 25 favourites, one per Company graded great or ok, gave concordance of 62% at β 0, 66% at 0.2, 71% at 0.4, 77% at 0.6, 82% at 0.8 and 85% at 1.2, and positives in the top 20 of 8, 9, 9, 11, 13 and 13. Those gains are inflated, since the favourites were picked from the same Grades, and only one `no` Job sat at a favourite Company (hunter-bond, rank 43 at β 0, 28 at 0.4, 20 at 0.8). At 1.2, nine Jobs tie at 98. The data can't separate 0.4 from higher values, so the plan's starting value stands. Re-run `just eval-scoring` once real favourites exist and raise β only if positives at favourite Companies still rank below their Grades.

## Amendment: missing evidence counts as a coin flip (α)

A nice or ok dimension the judge couldn't read used to drop out of both sides. A sparse ad that matched role, seniority, tech and work and said nothing about domain, stage or size scored `(9 + 0.5) / (9 + 1) = 95`, so 95 only meant "everything we know matches", and 78 of 380 open Jobs were Great.

Each picked dimension now also adds `missing = α * (1 - evidence)` at half credit:

```
numerator   += w * (evidence * credit + 0.5 * missing)
denominator += w * (evidence + missing)
```

The same sparse ad scores `(9 + 2 + 0.5) / (9 + 4 + 1) = 82`. A dimension with full evidence scores as before, and a dimension with no Picks still counts on neither side. `α` is `missingAlpha` in `compute.go`, set to 1 by hand; #623 fits it against Grades.

Replay for the one labelled user (5 positives, 3 negatives): Great Jobs went from 78 to 56 of 380, the top score from 95 (4 tied) to 93 (1), and positives in the top 20 from 3 to 4. Median positive rank percentile moved from 0 to 4, since the top-tied positives now spread out. The mid-level negative (volition) still ties the lowest-scored senior positives at 86. Band cut-offs are unchanged; #623 recalibrates them on this distribution.
