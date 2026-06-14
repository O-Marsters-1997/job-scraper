# ADR 0005 — Phase-agnostic score-on-ingest and a universal relevance gate

**Status:** Accepted

## Context

The pipeline has two decoupled phases: **Crawl** (discover + enqueue URLs) and **Enrich** (dequeue + fetch detail + upsert). Two new capabilities have to slot in:

1. **Relevance gate** — a cheap heuristic filter that drops irrelevant candidates *before* the next expensive stage.
2. **Suitability scoring** — a per-job Claude Haiku call producing a 0–100 fit score.

The original PRD placed the gate in Crawl (to protect the detail fetch) and suitability scoring "in the worker ingest path after `db.Save`" (i.e. in Enrich). But ATS API Sources complete entirely in Crawl — they have no Enrich phase (`NeedsDetail() == false`, full details returned by the list call). That produces two contradictions:

- An API-sourced Job would be saved in Crawl and **never scored**, because scoring lived in Enrich.
- The gate's stated job — "protect the expensive detail fetch" — is meaningless for API Sources, whose fetch is free. Yet the per-job LLM call is now the expensive stage for those Jobs, and an ATS board can return hundreds of irrelevant roles.

## Decision

**Decouple suitability scoring from Enrich into a phase-agnostic score-on-ingest step** invoked at the moment of persistence, called by *both* the API-source Crawl path and the HTML Enrich path. Every saved Job is scored exactly once, regardless of which phase saved it.

**Generalise the relevance gate to run pre-persistence for all Sources.** Its purpose shifts from "protect the detail fetch" to "protect whatever the next expensive stage is":

- HTML Source: gate runs in Crawl, pre-enqueue — protects the detail fetch.
- ATS API Source: gate runs after the (free) board fetch, before Save + scoring — protects the LLM call and the DB write.

The same `Scorer` (heuristic, v1) is applied at the right point per Source type.

## Consequences

- Scoring is no longer "part of Enrich"; `CONTEXT.md`'s Crawl/Enrich entry notes that API Sources complete in Crawl alone, and **Suitability** is defined as a post-persistence step independent of phase.
- **Implementation ripple (flagged, not yet decided):** the relevance score is computed during Crawl, but for HTML Sources the Job row does not exist until Enrich. The computed **Relevance** must therefore be carried through the queue alongside the URL, or recomputed at Enrich. The queue today stores a bare URL as the sorted-set member, so carrying the score requires a richer queue payload. This is left to implementation.
- LLM cost scales with *relevant* jobs, not board size, because the gate runs before scoring on every Source.
- See ADR 0006 for where the resulting scores are persisted (per-user, not on the shared Job).
