---
name: tune-tailoring
description: Change the CV-tailoring prompts, checks or model safely, and add eval fixtures. Use for edits to voice.md, rules.md, internal/services/tailoring/checks, cvedit.Model, or "add a tailoring fixture". Requires an eval run before and after.
paths: ["internal/services/tailoring/cvedit/**", "internal/services/tailoring/checks/**", "internal/services/tailoring/eval/**", "cmd/eval-tailoring/**"]
---

# CV tailoring

Design and trade-offs: [ADR 0013](../../../docs/adr/0013-cv-tailoring.md). Sonnet edits slots from
the Experience Bank, and deterministic checks gate the result. Prompts and checks are tuned per
model, so they change together or not at all.

## Changing voice.md, rules.md, a check or Model

1. Export `OPENROUTER_API_KEY` and run `just eval-tailoring -runs 5`. Keep the output: this is the baseline.
   No Google credentials are needed; fixtures carry the parsed base CV.
2. Make the change. `cvedit.PromptVersion` is the hash of `voice.md` and `rules.md`, so a prompt
   edit gives a new `prompt_version` automatically. A check or `Model` change does not, so state it
   in the PR.
3. Run the same command again. Compare each check's "first attempt" and "after retries" rates and
   the retry counts per fixture against the baseline.
4. Put both tables in the PR description. Do not merge a change that lowers a pass rate or raises
   retries without saying why.
5. A new check needs unit tests in `internal/services/tailoring/checks` and its name added to `checkNames`
   in `internal/services/tailoring/eval/eval.go`.

`voice.md` and `rules.md` are developer-owned today. If users later supply their own voice, `rules.md` splits
into a fixed contract and user preferences, and the eval should then check that grounding and the other
checks hold under any voice.

Runs cost real money and vary, so use `-runs 5` or more before drawing conclusions and `-fixture NAME`
to iterate on one scenario.

## Adding a fixture

A fixture is `internal/services/tailoring/eval/fixtures/<name>.json`, loaded by `eval.Fixtures`:

- `Scenario`: the PRD scenario or failure it probes.
- `JobDescription`: plain text.
- `Positions`: Bank Positions with `ID`, `Employer`, `Title`, `Achievements` (`ID`, `Text`) and
  `HeadingIndex`, the index into `Structure.Headings` of the role heading they sit under.
- `Structure`: a `docparse.DocStructure` as JSON (Go field names). Slots whose `HeadingIndex` matches
  a Position are that Position's bullet slots; include `Profile` and `Skills` to exercise them.

Better still, capture `Structure` from a real CV with `docparse.Parse`. Never put real personal
data in a fixture. Keep the four PRD scenarios covered: a job skill missing from the Bank (gap),
a backend re-ranking, a temptation to embellish numbers, and tight slot lengths.
