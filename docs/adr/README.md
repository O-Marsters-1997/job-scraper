# Architecture decision records

Numbering is not contiguous. Two numbers have no file:

- **0023** is cited as `0023-answers-and-scores-are-per-user.md` and was never committed. The rule it
  records (answers and scores are per user) is described in ADR 0011 and `CONTEXT.md`. Restoring it is
  tracked in #823.
- **0024** is cited as `0024-hostile-sources-fetch-residential-only.md` in `internal/worker/proxy/fetch.go`
  and was never committed. Its decision is folded into ADR 0018.

Do not reuse either number. The API-polls ADR that collided with 0025 is now 0027.

Where a global skill (go-idiomatic, go-testing, postgres-best-practices) conflicts with an ADR, the skill
wins and the ADR is amended in place (AGENTS.md).
