---
name: tune-tailoring
description: Change the inline-suggestion prompts, checks or model safely, and add eval fixtures. Use for edits to voice.md, suggest.md, internal/services/cvtailor/checks, cvedit.SuggestModel, or "add a tailoring fixture". Requires an eval run before and after.
---

# CV tailoring

Design and trade-offs: [ADR 0013](../../../docs/adr/0013-cv-tailoring.md). A Draft is built without a
model; the model only writes inline suggestions on one line at a time (`cvedit.Suggest`), and the
deterministic checks flag the result. Prompts and checks are tuned per model, so they change together
or not at all.

## Changing voice.md, suggest.md, a check or SuggestModel

1. Export `OPENROUTER_API_KEY` and run `just eval-tailoring -runs 5`. Keep the output: this is the baseline.
   No Google credentials are needed; fixtures carry the line and its Achievements.
2. Make the change. `cvedit.SuggestPromptVersion` is the hash of `voice.md` and `suggest.md`, so a
   prompt edit gives a new `prompt_version` automatically. A check or `SuggestModel` change does not,
   so state it in the PR.
3. Run the same command again. Compare the overall pass rate and each fixture's rate against the
   baseline, and read every FAIL line.
4. Put both outputs in the PR description. Do not merge a change that lowers a pass rate without
   saying why.
5. A new check needs unit tests in `internal/services/cvtailor/checks`. Wire it into
   `eval.RunSuggest` if suggestions should be scored on it.

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
- `Action`: `fit`, `tighten`, `verb` or `ask`; `MaxChars` applies to `fit`, `Prompt` to `ask`.

Never put real personal data in a fixture. Keep every action and the embellishment temptations
(an invented metric, an invented technology) covered.
