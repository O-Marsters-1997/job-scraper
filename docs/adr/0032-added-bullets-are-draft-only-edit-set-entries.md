# ADR 0032 — Added bullets are draft-only entries in the edit set

Extends [ADR 0019](0019-in-app-page-edits-google-renders-record.md).

## Context

A Draft bullet was always a slot of the base CV, so the editor could not add or remove one (#853). A bullet
the User adds has no base paragraph and no slot ID.

## Decision

- **Identity.** The client mints a `new-<uuid>` slot ID. The server accepts any `new-` ID and never reuses
  base IDs for it.
- **Storage.** `PositionEdit.Added` holds `{slotId, after, text}` beside the base `Bullets`, which keep
  their positional mapping to base slots. `after` names the bullet it follows, a base slot or another
  added one. `PositionEdit.Lines()` is the one place that merges both into page order.
- **Save.** `PUT .../slots` takes `{slotId, text, after}` to add or update an added bullet and
  `{slotId, remove: true}` to delete one. Removal is idempotent and re-anchors its followers. A `slots`
  removal of a base bullet is rejected: only what the User added can be removed that way.
- **Exclusion** (amended for #1449). A role's `positions` list may omit base bullets: `docedit` deletes the
  Doc paragraphs left without a line. `GET .../roles` returns the omitted ones as `excluded`, so a client
  can show them dimmed and put them back by resending their text in the list. A base bullet swapped out
  for another is not excluded; it stays under that bullet's `replacedFrom`. The list still keeps at least
  one bullet.
- **Rendering.** `docedit` maps the merged lines onto the Draft Doc's paragraphs by position. Extra lines
  are inserted as `"\n" + text` at the end of the position's last paragraph, so Google copies that
  paragraph's bullet and paragraph style. Newly inserted text takes the preceding character's text style.
- **Checks.** Added bullets are the User's own words with no cited Achievements and no base text. They run
  through the same checks as an edited bullet, so a number or term the Position's Achievements don't
  support is flagged. Findings never block a save. The length check has nothing to compare against and
  skips them, and the page count check sees them through the Google export.
- **Editor.** Enter at the end of a non-empty bullet adds an empty line below it. Backspace on an empty
  added line removes it. An empty added line is never saved, because the Doc cannot hold an empty bullet
  (`docparse` skips empty paragraphs), so it disappears on reload.
- **Undo and Changes.** Undo on an added bullet removes it. Changes lists it as Added through the
  existing bullet diff.

## Consequences

- Draft Docs made before this change load unchanged, since `Added` is optional.
