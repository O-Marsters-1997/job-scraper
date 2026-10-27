# ADR 0038 — The Profile is redrafted by a model, tuned against the user's own edits

## Context

A Draft was built without a model: bullets are Achievements picked by Jev lean, Skills come from the CV and
Bank, and a model only rewrites one line when asked (the inline suggestions). The Profile stayed as written
in the base CV, so it never addressed the Job. A paragraph that addresses the Job needs real rewriting,
which picking can't do. That rewriting has to stay grounded and cheap, and the user has to be able to see
whether it is getting better.

## Decision

- **A Profile Redraft runs only on request.** Generation never touches the Profile, so a Draft is still
  built without a model writing its text. The editor's Redraft action works from the base Profile with an
  optional one-line steer, saves the accepted text into the edit set (undo restores the previous text) and
  returns the complete text, without streaming. It works the same on `generate` and `as_is` Drafts.
- **The input is minimal.** Base Profile, Bank Skill names, Job title and company, and the Job description
  stripped of HTML and capped. No Achievements. The prompt is ordered static rules, then the Job, then the
  Profile, so prompt caching can be added later without changing it.
- **Guardrails, then fallback.** The output must pass Profile grounding (Achievements and Bank Skills),
  banned words, and length no longer than the base. On a violation it gets one retry with the findings fed
  back. If that also fails, or the call errors, the Profile is left as it was and the user sees why. There
  is no cap on changed words; voice is judged in the evals.
- **Every Attempt is captured.** `profile_attempts` holds each generated Profile with its steer, prompt key,
  model, cost and any rejection. The target is the kept Draft's final Profile, read from its edit set.
- **Evals score edit burden.** One package (`cvtailor/profiledraft`) owns the prompt, model and checks, and
  both production and `cmd/eval-tailoring` call it. Committed fixtures are synthetic and cover guardrails.
  An export turns captured Attempts into fixtures in a gitignored local directory, scored by the word-level
  distance from the new output to the user's kept final.

Rejected: redrafting during generation (bulk Drafts would arrive with a Profile ready, but every Draft
would pay for a paragraph the user may never want changed, and model text would land unreviewed); few-shot
prompting with past edits (adds tokens to every call, and behaviour changes with no eval
gate); an LLM judge (cost per run, and a second prompt that drifts); committing anonymised real fixtures
(anonymising breaks grounding cases, and leaks are possible).

## Consequences

- Each Redraft click spends up to two Sonnet calls on the user's OpenRouter key. Generation spends none.
- Edit-burden scores exist only on a machine holding exported captures; CI sees only the guardrail rate.
- A Profile reverted to base is a negative signal: its final equals the base.
