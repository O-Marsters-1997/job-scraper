---
name: tune-tailoring
description: Change the inline-suggestion prompts, checks or model safely, and add eval fixtures. Use for edits to voice.md, lineedit/suggest.md, the lineedit action table or checks, internal/services/cvtailor/checks, or "add a tailoring fixture". Requires an eval run before and after.
---

# CV tailoring

Design and trade-offs: [ADR 0017](../../../docs/adr/0017-app-owned-drafts-google-rendered.md). A Draft is
built without a model. A model only writes text when the user asks, either an inline suggestion on one line
or a Profile Redraft. The redraft goes through `cvtailor/profiledraft`
([ADR 0038](../../../docs/adr/0038-profile-redraft-by-model-tuned-on-attempts.md)), and the same
before-and-after eval rule applies to it. Every suggestion,
in production and in the eval, goes through `lineedit.Editor.Rewrite`
(`internal/services/cvtailor/lineedit`), which owns the action table, `suggest.md`, the model, the
checks and the length rules. `voice.md` stays in `cvedit` as `cvedit.Voice`. Prompts and checks are
tuned per model, so they change together or not at all.

## Changing voice.md, suggest.md, an action, a check or the model

1. Export `OPENROUTER_API_KEY` and run `just eval-tailoring -runs 5`. Keep the output: this is the baseline.
   No Google credentials are needed; fixtures carry the line and its Achievements.
2. Make the change. The report prints each fixture's `prompt_key`, the hash of its model, system and
   user prompts, so editing `voice.md`, `suggest.md`, an action's wording or the model changes it. A
   check or length-rule change does not, so state it in the PR.
3. Run the same command again. Compare the overall pass rate and each fixture's rate against the
   baseline, and read every FAIL line.
4. Put both outputs in the PR description. Do not merge a change that lowers a pass rate without
   saying why.
5. A new check needs unit tests in `internal/services/cvtailor/checks`. Add it to `findings` in
   `lineedit`, so production and the eval both run it.

Never run `just eval-tailoring`: it spends real money on the user's OpenRouter key and Claude Code is
denied it. When a change needs a baseline, give the user the command with the `-runs` you intend and
let them run it (`! just eval-tailoring -runs 5`), then work from the output they paste back.

Runs cost real money and vary, so use `-runs 5` or more before drawing conclusions and `-fixture NAME`
to iterate on one scenario.

## Adding a fixture

A fixture is `internal/services/cvtailor/eval/fixtures/suggest/<name>.json`, loaded by
`eval.SuggestFixtures`:

- `Scenario`: the failure it probes.
- `Text`: the line being edited.
- `Achievements`: the Achievement texts the suggestion may draw on.
- `Scope`, `Bank` (optional): the Position's Achievements and every Achievement the user has; each
  defaults to `Achievements`. `Profile` (optional) marks a profile paragraph, checked against `Bank`.
- `Action`: `fit`, `tighten`, `verb`, `ask` or `tailor`; `MaxChars` applies to `fit` and `tailor`,
  `Prompt` to `ask` and `Quote` to `tailor`.

Never put real personal data in a fixture. Keep every action and the embellishment temptations
(an invented metric, an invented technology) covered.
