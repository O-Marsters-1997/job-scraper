# ADR 0002 — Google Docs CVs via server-side PDF export

A CV is one **Tab** of a Tracked Google Doc, rendered in-app from a server-side Drive PDF export (pdf.js), not rebuilt from Docs API content. Only Google's own export reproduces its pagination exactly, and a one-page CV must stay one page. Per-tab export uses an undocumented `tab` parameter, falling back to the whole-doc PDF opened at the tab.

- The OAuth scope is `drive.readonly`, a restricted scope. The GCP app stays in **Testing** with named test users rather than pass Google's security assessment. That is fine for the owner plus a small circle; going public requires the assessment.
- Google tokens are encrypted at rest (AES-256-GCM, `GOOGLE_TOKEN_ENC_KEY`), because a leaked token exposes the user's whole Drive.
- Tab titles and content are read live. The only persisted Tab state is a `visible` flag (`tracked_doc_tabs`), so a user can hide a Tab without untracking the doc. Rows are reconciled on each list, so new Google tabs appear automatically.
