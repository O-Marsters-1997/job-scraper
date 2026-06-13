## Approach

- Think before acting. Read existing files before writing code.
- Be concise in output but thorough in reasoning.
- Prefer editing over rewriting whole files.
- Do not re-read files you have already read unless the file may have changed.
- Test your code before declaring done.
- No sycophantic openers or closing fluff.
- Only add comments if code is NOT self-descriptive.
- Keep solutions simple and direct.
- User instructions always override this file.

## Project decisions

When making structural or architectural decisions — refactoring package boundaries, choosing where a new type or layer belongs, or resolving naming ambiguity — consult if they exist. Not for routine implementation.

- `CONTEXT.md` — domain language glossary
- `docs/adr/` — architectural decisions

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`O-Marsters-1997/job-scraper`). See `docs/agents/issue-tracker.md`.

### Triage labels

Default label vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context repo — one `CONTEXT.md` + `docs/adr/` at the root. See `docs/agents/domain.md`.

## UI / design

When implementing or changing frontend UI, read `DESIGN.md` (repo root) first — it is the source of truth for colours, typography, spacing, components, and voice. Use its tokens and patterns; do not invent values outside its palette.

- Before adding a new UI pattern, check whether `DESIGN.md` already defines one.
