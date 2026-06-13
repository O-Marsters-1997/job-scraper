# Google Docs CVs via server-side PDF export

CVs are rendered in-app by exporting the source Google Doc to PDF server-side (Drive API, using the user's OAuth token) and rendering that PDF with pdf.js, rather than fetching the structured Docs API content and rendering it ourselves. The requirement is exact page fidelity — a CV that fits one page in Google Docs must look like one page in the app — and only the PDF export reproduces Google's own pagination, spacing, and page breaks. A custom Docs-to-HTML renderer was rejected because reproducing page-fit fidelity that way is prohibitively expensive and still approximate.

Each Google Docs **Tab** is treated as one CV. Drive's documented PDF export is whole-doc, so single-tab isolation uses the undocumented export endpoint with a `tab` parameter, falling back to a whole-doc PDF opened at the tab when that does not isolate. This fallback is the reason the feature is resilient to Google changing or never supporting per-tab export.

The OAuth scope is `drive.readonly` — a Google "restricted" scope. Rather than complete Google's restricted-scope security assessment, the GCP OAuth app stays in **Testing** publishing status with named test users. This is acceptable only while the user base is the owner plus a small circle; going public later requires the assessment. This constraint is deliberate and recorded here because a future reader will otherwise wonder why the app isn't published.

Google tokens are encrypted at rest with AES-256-GCM (key from `GOOGLE_TOKEN_ENC_KEY`), because the feature is intended to be shared with other users and a `drive.readonly` token grants read of a user's entire Drive — a plaintext database leak would expose every linked user's Drive, not just the owner's.

See PRD #33 and `CONTEXT.md` for the **Tracked Doc** / **CV** / **Tab** / **Google Link** vocabulary.
