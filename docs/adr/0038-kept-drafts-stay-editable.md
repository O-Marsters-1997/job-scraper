# ADR 0038 — Kept Drafts stay editable

Status: accepted. Agreed in the iOS feedback session (#1450). Supersedes the "kept is read-only" rule in
`CONTEXT.md` (Draft) and amends [ADR 0017](0017-app-owned-drafts-google-rendered.md).

## Context

Keeping a Draft froze it: iOS and the web editor stopped offering edits once `outcome = 'kept'`, though
`SaveDraftSlots` only ever refused drafts that weren't ready or were discarded. Keep today renames the
render-cache Doc and marks it `kept_as = 'doc'`; it no longer builds a child Tab. Users found a typo or a
better bullet after keeping and had to discard and redraft to fix it.

Keep also records one `achievement_outcomes` row per bank Achievement (#1236). An edit after keep would
leave those rows describing a CV that no longer exists.

## Decision

- **Keeping commits a Draft, it doesn't freeze it.** A kept Draft is edited exactly like a ready one, on
  iOS and web, and each save writes its kept Doc. Only a discarded Draft is read-only.
- **A save on a kept Draft re-records its outcomes.** After the edit is saved, `SaveDraftSlots`
  recomputes them with the same `keepOutcomes` the keep tick uses and replaces the stored rows in one
  transaction. If that step fails the edit stays saved and the save reports the error.
- Keep stays one-shot: a kept Draft can't be kept again, and Apply stays available after an edit.

## Consequences

Outcomes always describe the CV the user sent. Each save of a kept Draft costs one extra Jev call for the
bank's answers; a Jev failure still records outcomes without leans, as keep does. Recorded outcomes no
longer mark the moment of keeping, only the latest kept content.
