# ADR 0006 — Per-user scoping of Search Config and Job scores

**Status:** Accepted

## Context

Relevance and Suitability are scores of a Job *against a specific user's criteria and rubric*. The PRD originally proposed storing `relevance_score` and `suitability_score` as columns on the `jobs` table and holding the search criteria in a single global-row `search_config`.

But `jobs` is deliberately a **shared catalog**: it has no `user_id`, and a User's relationship to a Job is already modelled by **Application** (`UNIQUE (user_id, job_id)`). Every other table in the app — `applications`, `application_statuses`, `tracked_docs`, `google_oauth_tokens`, `sessions` — is per-user with a `user_id` FK to `users`. Storing scores on the shared `jobs` table would:

- make scores global/single-user on a table that is otherwise user-agnostic by design;
- contradict the extensibility principle ("every scoring abstraction must accept criteria as an injected dependency, not a hardcoded value");
- require migrating scores *off* `jobs` the moment a second intent/user is added.

A score is not a property of a Job — it is a Job↔User assessment, structurally identical to Application.

v1 is explicitly single-recipient/single-intent. The question is whether to model that as *global data* or as *per-user data with one seeded user*.

## Decision

Scope both Search Config and scores **per user**, even in v1:

- **`search_config`** carries `user_id` (UNIQUE per user) and holds role/location/keywords, the suitability rubric, the relevance cutoff, and the notify threshold. v1 seeds exactly one row for the current user.
- **Scores live in a per-user/per-job table** (e.g. `job_scores` with `UNIQUE (job_id, user_id)`) holding `relevance_score` and `suitability_score`. The `jobs` table is left unchanged as a shared catalog.

v1 behaviour stays single-tenant (one seeded user, one config), but the data model matches the schema grain and needs no future migration to un-bake single-user.

## Consequences

- The jobs-list query (`GET /jobs`) joins `job_scores` for the current user to surface Relevance/Suitability columns; jobs with no score row yet sort accordingly.
- The relevance gate, suitability scorer, and notification gate all read the current user's `search_config` as an injected dependency — never a constant.
- `description` and `salary_raw` remain on `jobs` (they *are* properties of the shared Job — see ADR 0007); only the per-user *assessments* move to `job_scores`.
- Multi-user later is a behavioural change (seed more users, fan out scoring), not a schema migration. The known future cost is that ingesting one Job implies one scoring call per user; out of scope for v1.
- `CONTEXT.md` relationships record: a Job carries a Relevance and Suitability score *per User*, sibling to Application; a User has exactly one Search Config.
