# ADR 0017 — App-owned Drafts, rendered by Google, kept as verified child Tabs

Amended by [ADR 0019](0019-in-app-page-edits-google-renders-record.md): the in-app page is the editing
surface and an instant estimate. Google stays the renderer of record. The "Rejected: our own renderer"
paragraph and the first consequence below are superseded.

## Context

ADR 0002 makes a CV a Google Docs Tab rendered by Drive's PDF export, because only Google's export
reproduces its own pagination. Drafts followed suit: each Draft is a copied Doc and that Doc is its
source of truth. An in-app editor with a live preview needs the app to own the content, but any
renderer of our own drifts from Google's line breaks, and a one-page CV must stay one page.

The Docs API constrains the keep step: `addDocumentTab` creates only an empty Tab, and nothing copies a
Tab or its content between or within Docs.

## Decision

- **The app owns a Draft.** `tailored_cvs.edit_set` is the source of truth. The Draft's Doc is a hidden
  render cache, re-synced from `edit_set` a section at a time (Profile, Skills, one Position's bullets)
  and exported for every preview. Google remains the only renderer; ADR 0002 still holds for every PDF.
- **Keep rebuilds the Draft as a child Tab under its base CV**, from the render-cache Doc's structure
  with every paragraph and text style written explicitly, since the API cannot set a Tab's named
  styles. This needs the `documents` scope (sensitive, not restricted), requested incrementally.
- **The kept Tab is verified, not trusted.** Its PDF export is compared with the preview by glyph run
  (text, page, x/y within 0.5pt, font, size). On any mismatch the Tab is deleted and the render-cache
  Doc is kept as a standalone Doc instead, with the reason shown.
- **Keep runs on the cvtailor tick**, not in the request, so a half-built Tab is always cleaned up.

Rejected: our own renderer with a verify-on-keep check. It previews instantly, but "close" isn't
"identical", and the user's constraint was that formatting never changes. Copying the base Doc on every
sync: simpler, but slower and it leaks base-CV edits made mid-Draft into the Draft.

## Consequences

- A preview costs one Docs get, one `batchUpdate` and one export (2–5s), so it refreshes after a typing
  pause, not per keystroke.
- A kept Draft is a child Tab, so `ListTabs` (top-level only) keeps it out of the CV library and it can
  never be the base of another Draft.
- Base CVs using images, tables or columns may fail verification and land as standalone Docs until the
  rebuild learns those elements.
