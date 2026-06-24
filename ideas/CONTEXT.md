---
last_updated: 2026-06-24
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

## First-principles insights

- **Eureka 1 — discarded explanation.** Suitability LLM already runs against the rubric at ingest,
  but the *reasoning* is thrown away and the rubric is uneditable in the UI. The exact gap OSS ATS
  roadmaps name ("show why it scored that way") is one schema change away. Reuse, not new spend.
- **Eureka 2 — variant selection beats variant generation.** Incumbents generate/edit one resume.
  This repo stores N CV variants per user → pick the best per job + show the gap, no generation needed for v1.

## Implemented ideas

(Feature work shipped per git history — scraper/enrichment side mostly out of ideation scope.)
- Score-ranked jobs API + frontend score columns (#60)
- SuitabilityScorer + score-on-ingest (#56); suitability-gated notifications + digest filter (#59)
- RelevanceScorer + WIS card parsing (#52); relevance gate (#54)
- job_scores + search_config tables + `GET/PUT /search-config` API (#50, #53)
- Per-user source targets, DB-driven source registration; source selector in tracked searches
- Authenticated UI hardening + violet design system consolidation
- Detail enrichment: Description, SalaryRaw, WorkArrangement (scraper side)

## Proposed ideas (pending)

- [2026-06-24] Glass-box scoring — capture suitability reasoning/matched/missing + surface the
  editable rubric & thresholds (frontend already lacks the existing /search-config controls). **(Recommended next.)**
- [2026-06-24] AI settings — per-user model picker, default-cheap (enabler).
- [2026-06-24] Score-validated funnel analytics — does the score predict responses?
- [2026-06-24] Best-CV-for-this-job — rank existing CV variants + gap analysis.
- [2026-06-24] Tailored application draft — cover letter / bullets from JD × chosen CV.
- [2026-06-24] Market skills-gap radar — portfolio-level skills missing from all your CVs (`[enrich]`).
- [2026-06-24] Salary intelligence — normalise salary_raw, benchmark offers.
- [2026-06-24] Application Copilot (moonshot) — scored job → matched CV → drafted letter → tracked daily loop.

See `ideas/reports/2026-06-24-ideate.md` for full detail.
