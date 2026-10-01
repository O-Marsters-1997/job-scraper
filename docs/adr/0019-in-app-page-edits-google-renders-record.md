# ADR 0019 — The in-app page is the editing surface; Google is the renderer of record

Amends [ADR 0017](0017-app-owned-drafts-google-rendered.md).

## Context

ADR 0017 rejected a renderer of our own because "close" isn't "identical". That kept the Draft page a
PDF iframe beside a list of bullet cards, and the cost showed. The CV fills its page, so one wrapped
line decides between one page and two, and you learn that only after a save and a 2-5s export.

An in-app page at true size, in pt-accurate type, shows the line count as you type. It can't be
identical to Google's output, but it can be measured against it.

## Decision

- **The in-app page is the editing surface and an instant estimate.** The user edits text on a page
  laid out from the Draft Doc's own styles, with line counts and findings in margin cards.
- **Google stays the renderer of record.** Every save still runs `ExportPDF`, `pageCounts` and
  `checks.PageCount`. Print preview, Keep and the kept Tab come from Google's export only. Keep
  verification still compares Google with Google.
- **The page meter defers to Google.** When the estimate and Google's page count disagree, the meter
  shows Google's ("Google renders 2 pages"). Only Google's PDF is ever shown as final.
- **Saves stay on `PUT /tailoring/drafts/{id}/slots`.** The client serialises them, one in flight and
  one pending, each carrying every editable slot, so a repeated save is idempotent. This is safe for
  one tab. Two tabs on one Draft can lose an edit until the revision check in #517 lands.

ADR 0017's "Rejected: our own renderer" paragraph and its first consequence (preview refreshes after a
typing pause) no longer hold. App-owned content, the Doc as render cache, and keep as a verified child
Tab stand.

## Consequences

- The estimate is only as good as its fonts and metrics, so ADR 0020 pins both and a calibration
  fixture fails CI when they drift from a real Google export.
- Tables, columns, images and headers or footers aren't laid out by the page. Such a Draft returns 422
  and falls back to the PDF view.
- A new `GET /tailoring/drafts/{id}/layout` reads the Doc live on each open, with no cache column and
  no schema change.
