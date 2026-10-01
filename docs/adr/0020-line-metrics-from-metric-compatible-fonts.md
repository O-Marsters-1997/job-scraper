# ADR 0020 — Line metrics from metric-compatible web fonts

## Context

The in-app page (ADR 0019) has to predict where Google wraps a line and how tall the line box is. A
CV that fills its page has no slack. The estimate must land within about 1pt per block of Google's PDF.

Docs fonts such as Calibri and Arial are licensed and not available to the browser. Google substitutes
nothing visible, because its export embeds the real font.

## Decision

- **Line box height is computed from font tables.** `size x (hhea ascender + descender + lineGap) /
  unitsPerEm x lineSpacing/100`, using values read from the font file.
- **Docs fonts map to self-hosted metric-compatible fonts:** Calibri to Carlito, Arial to Arimo, Times
  New Roman to Tinos, Cambria to Caladea. Same advance widths, so the same line breaks.
- **Unknown fonts fall back to one generic metric font**, and the layout response carries a "line breaks
  may differ" warning that the page shows.
- **A calibration fixture guards it.** It compares the renderer's line counts and block heights with a
  real Google export and fails CI on drift.

Rejected: per-keystroke server layout through a PDF export (2-5s, the problem we are leaving), canvas
text measurement (a second layout engine to keep in step with the browser's), and loading Google Fonts
CSS at runtime (a third-party request on every open, and no control over the font version).

## Consequences

- Fonts ship with the frontend bundle. A new Docs font in a base CV needs a mapping, or it gets the
  warning.
- Spacing, indents and borders come from the Doc's paragraph styles via the layout route, so the
  renderer never guesses them.
