# ADR 0017 — Company-level tracking across verified ATS Boards

**Status:** Accepted; implementation pending
**Partially supersedes:** ADR 0016 (tracking identity and check frequency)

## Context

ADR 0016 deliberately made an enabled ATS Source Target the definition of a Tracked Company. That kept one source of truth while the catalog supported one ATS Board per Company. It also meant a User could not track a Company until its Board was known. A Company with several regional or provider-specific Boards would need several independent targets, and later-discovered Boards would not inherit the User's existing interest.

The product intent is to select Companies of interest, then check their Boards cheaply and reuse the resulting Jobs across Users. The User's interest is in the Company, not a particular board token. There are genuine costs to separating tracking from Source Targets: a new relation and a shared polling state need coordination, and the current query/builder/UI path must change. Keeping board-level targets would preserve existing code but make tracking incomplete as Companies add or change Boards.

## Decision

- A User tracks a Company once, including when no ATS Board is known yet. The choice covers every current and later **verified** Board associated with that Company. Discovery of a Board starts collection for interested Users without another approval step.
- Periodic YC/Getro harvesting adds Companies to the shared catalog and surfaces newly discovered Companies in the Companies view. It does not inspect the careers sites of untracked Companies to find ATS Boards. A User may attach a Board to a Company from that view.
- A Company may have several verified ATS Boards. A direct ATS Board URL identifies or creates its Company, attaches the verified Board, and then follows the same company-level tracking path. It does not create an independent board-only tracking concept.
- Verification requires a successful ATS Board fetch plus evidence of Company association: a link from the Company's careers site or explicit User confirmation. Merely detecting an ATS-shaped URL is insufficient to start automatic polling.
- Inspect a Company's careers site when a User begins tracking it, then recheck about monthly while tracked to find changed or additional Boards. Recheck sooner after repeated Board failures. A reachable but obsolete Board must not prevent discovery of its replacement.
- When a careers site replaces an old Board with a new verified Board, begin checking the new Board immediately. Continue checking the old Board until two successful complete empty checks close its remaining Jobs, then retire the old Board from polling.
- A User chooses one Check Frequency per Tracked Company. It applies to all that Company's Boards. When several Users track the same Company at different frequencies, each Board is checked once at the shortest requested interval; results are shared.
- Discovery Source Targets remain User-defined searches. Shared Board polling state is distinct from user interest and must support one active claim per Board when several workers run.

## Consequences

- Company tracking survives missing, newly discovered, additional, and migrated ATS Boards. The catalog must distinguish Company identity from board identity and verify associations before polling them.
- Careers-site revalidation is restricted to tracked Companies at a low cadence, with failure-triggered checks, rather than repeatedly crawling every catalog entry.
- The current crawler's automatic inspection of untracked catalog Companies is outside this design and must be changed; merely appearing in the catalog does not cause careers-site fetching.
- The current `companies.ats_source`/`ats_token` pair, board-specific ATS Source Targets, and per-target `last_checked_at` do not implement this decision. The implementation needs a Company-to-Board association, per-User company tracking and frequency, and shared Board polling state. Existing code remains unchanged by this ADR.
- Automatic merging of a vacancy across different ATS providers after a migration is deferred. Different provider posting IDs remain different Jobs unless a later, explicit identity rule is agreed; the short application window does not justify a matching workflow now.
- The shortest requested interval can increase shared fetch cost for all subscribers. One fetch per Board and explicit provider budgets limit duplication; UI should show the User's requested frequency without implying a separate fetch per User.
- ADR 0016 still governs the shared catalog and interest-gated suitability scoring in principle. Its claim that tracking *is* an ATS Source Target, its one-Board Company ceiling, and its per-target frequency ownership are superseded here. Scoring fanout should follow company-level interest for ATS Jobs after migration.
