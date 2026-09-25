---
last_updated: 2026-07-01
---

# Ideation Context — Job Scraper

Canonical context for `/ideate` runs. Future runs start here instead of re-deriving.

## Positioning

**What:** A personal job-hunting command centre — Go API (`cmd/api`) + SolidStart frontend
(`frontend/`). Scrapes jobs, scores them **per-user** (heuristic *Relevance* + LLM *Suitability*
against a user rubric, `internal/score/`), tracks applications through **custom pipeline stages**,
and surfaces CVs **live as Google-Doc tabs** (PDF-exportable).

**Who:** Job seekers wanting a private, inspectable, self-owned alternative to Teal/Huntr. As of
2026-06-24 the product is being aimed at **more than one user** (multi-user polish is first-class).

**Thesis:** *Be selective and intelligent* — spend effort (scrape cost, LLM cost, user attention)
only where it pays off. Per-user scoring + a suitability gate already encode this.

**The moat — the unique asset:** almost no competitor holds all three at once:
1. Scored job corpus (`jobs` + per-user `job_scores`).
2. Real CVs as structured tabs — **multiple variants per user** (`tracked_doc_tabs`).
3. An LLM already wired in with a per-user suitability **rubric** (`search_config.suitability_rubric`).
The **jobs × CVs × LLM, per user** triangle is where exclusive value lives.

### Key data-model facts (for grounding ideas)
- `jobs`: title, location, url, company_slug, source, description, **salary_raw** (stored raw by
  design — ADR-0007), work_arrangement, scraped_at, updated_at.
- `job_scores`: per-(job,user) relevance_score + suitability_score (both nullable int). **No
  reasoning/explanation columns yet** — the LLM rationale is computed and discarded.
- `search_config` (1/user): role, location, keywords[], **suitability_rubric**, relevance_cutoff,
  notify_threshold. Backend API exists (`GET/PUT /search-config`, PR #53) but the **frontend never
  exposes the rubric/cutoff/threshold** — a pure frontend gap.
- `applications`: status_id, notes, applied_at, salary_info. `application_statuses`: custom stages.
- `tracked_docs` / `tracked_doc_tabs`: CV variants as Google-Doc tabs, PDF export wired.
- Rich structured fields (skills, salary range, days-in-office, experience level, company size) are
  **mocked in the frontend, absent in real data** — assumed to arrive via separate enrichment work.

### Settled decisions feeding ideation
- Multi-user is the target audience (not single-user-personal).
- Rich enrichment fields are assumed to become real — ideas may build on them (tag `[enrich]`).
- LLM features welcome, but must be **user-model-selectable, defaulting to a cheap model**.

## Competitive landscape

| Tool | Strength | Exploitable weakness |
|---|---|---|
| Teal | JD-paired resume builder; "honest" Match Score | Keyword-oriented; manual skills; builds resumes, doesn't hold your variants |
| Huntr | Semantic resume scoring + CRM; claims 2× interview rate | Score **overstates alignment** (false readiness); one resume, no variant library |
| Simplify | AI autofill copilot + pipeline tracking | Autofill, not fit reasoning; no corpus-level intelligence |
| JobSync / ApplyKit / ApplyPilot (OSS) | Self-hosted, data-private, BYO-LLM/Ollama | Single-applicant, single-resume; no "across my whole market" aggregate view |
| **career-ops** (OSS, ~56.7k★, added 2026-07-01) | Rubric-guided LLM scoring 1.0–5.0 **with citations to CV lines + JD reqs**; tailored PDF per role; drafts Greenhouse/Ashby/Lever answers; Go TUI | **Local, single-user, one-shot-per-JD, inside a coding CLI** — no standing corpus, no outcome memory, no CV **variant library**. Near-identical thesis; beatable only on hosted/multi-user/corpus-level ground |

**2026 market backdrop:** volume exploded to 300+/role → AI auto-apply backlash; employer ghosting
at 3-year high (53%); platforms suppress automation (~23% restricted in 90 days); per-role
**tailoring** is the one tactic proven to lift interview rate (~2% → 3–4%).

## First-principles insights

- **Eureka 1 — discarded explanation.** Suitability LLM already runs against the rubric at ingest,
  but the *reasoning* is thrown away and the rubric is uneditable in the UI. The exact gap OSS ATS
  roadmaps name ("show why it scored that way") is one schema change away. Reuse, not new spend.
- **Eureka 2 — variant selection beats variant generation.** Incumbents generate/edit one resume.
  This repo stores N CV variants per user → pick the best per job + show the gap, no generation needed for v1.
- **Eureka 3 (2026-07-01) — the market chased the wrong answer.** Everyone raced to *volume*
  (auto-apply, mass autofill); it produced workslop, ghosting and platform bans. First principles:
  the winning move is the opposite — **fewer, better, calibrated applications** — which is already
  this repo's thesis. Incumbents can't pivot here without becoming a different product.
- **Eureka 4 (2026-07-01) — nobody validates their own score.** Every competitor shows a match
  score; none feed *real outcomes* back to check it predicts responses. This repo holds `job_scores`
  **and** `applications` outcomes per user → can ship the only **calibrated, honest** score in the
  category, as read-only analytics over data already stored.

## Implemented ideas

(Feature work shipped per git history — scraper/enrichment side mostly out of ideation scope.)
- Score-ranked jobs API + frontend score columns (#60)
- SuitabilityScorer + score-on-ingest (#56); suitability-gated notifications + digest filter (#59)
- RelevanceScorer + WIS card parsing (#52); relevance gate (#54)
- job_scores + search_config tables + `GET/PUT /search-config` API (#50, #53)
- Per-user source targets, DB-driven source registration; source selector in tracked searches
- Authenticated UI hardening + violet design system consolidation
- Detail enrichment: Description, SalaryRaw, WorkArrangement (scraper side)
- **Glass-box scoring** (shipped 2026-06→07): `job_scores.reasoning`, `SuitabilityPanel`
  (score + rationale + matched/missing chips), on-demand "Explain score", editable rubric/cutoff/
  threshold in `scoring.tsx`. *Unshipped tail:* jobs-list row expander + below-cutoff toggle.
- **AI settings / model picker** (shipped): `settings/ai.tsx` — per-user scoring + reasoning model
  pickers (default Haiku 4.5), BYOK Anthropic key. *Note:* multi-tenant **ingest** still single-user
  (`SCORING_USER_ID`); `multi-tenant-byok.md` Phases 2–4 pending.
- LinkedIn source; BrightData Web Unlocker proxy; batch suitability scoring (Claude cache amortise).

## Accepted ideas

*Ideas you've decided to build, not yet shipped. `to-roadmap` boards this section when there is no
approach doc, so accepting is what puts an idea on the board.*

- [2026-09-25] BrightData proxy rollout — verify the existing BrightData Web Unlocker integration is working reliably, then let other sources opt into it as needed
- [2026-09-25] Observability with Grafana Cloud — ship structured logs, metrics and traces from the API, worker and queue to Grafana Cloud via OpenTelemetry, with dashboards and alerts for scrape runs, scoring and API errors

## Proposed ideas (pending)

- [2026-07-01] **Best-CV-for-this-job + real gap analysis** — fetch CV tab text (new
  `google.Client` method; `drive.readonly` already granted) → rank N variants vs JD → matched/missing
  vs the *chosen* CV. **(Recommended next — exercises the moat end-to-end; substrate for drafts + copilot.)**
- [2026-07-01] Calibrated "Honest Score" — plot response/interview rate per score bucket; the one
  claim no competitor can make (`job_scores` × `applications` outcomes). *(Differentiation flagship.)*
- [2026-07-01] Tailored draft — cover letter + Greenhouse/Ashby/Lever answers from JD × chosen CV.
- [2026-07-01] Company responsiveness & ghost-risk intelligence — cross-application company outcome memory.
- [2026-07-01] Paste-a-JD / bookmarklet capture (MVP, no extension) — manual door into ingest+score.
- [2026-07-01] Follow-up nudges for stalled applications — reuse notify/digest + status timestamps.
- [2026-07-01] Glass-box finish — jobs-list reasoning row expander + below-cutoff toggle.
- [2026-07-01] Selective Apply Copilot (moonshot) — hard-gated daily shortlist, best CV + draft + ghost-risk pre-assembled.
- [2026-06-24] Salary intelligence — normalise salary_raw, benchmark offers. *(still open)*

See `ideas/reports/2026-07-01-ideate.md` (latest) and `ideas/reports/2026-06-24-ideate.md`.
