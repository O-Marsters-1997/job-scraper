# ADR 0003 — Per-tab CV visibility

**Status:** Accepted  
**Supersedes:** ADR 0002 (partially — the "Tabs are never persisted" invariant no longer holds)

## Context

ADR 0002 stated that CV Tabs are "enumerated live from the Docs API, never persisted." That worked while the only removal action was deleting a whole Tracked Doc. The user now wants to hide individual Tabs without un-tracking the document; the list should show only visible Tabs.

## Decision

Introduce a `tracked_doc_tabs` table — one row per Google Docs Tab per Tracked Doc — storing a single `visible BOOLEAN NOT NULL DEFAULT TRUE` flag. This is the only field persisted about a Tab; all other attributes (title, content, modified-at) continue to be read live.

**Reconcile-on-list:** Each `List()` call runs `EnsureTabs` for every tracked doc — inserting default-visible rows for any live Google tabs that lack one (`ON CONFLICT DO NOTHING`). This means:

- Tabs created in Google after a doc was first tracked appear automatically on the next list load.
- Hidden choices survive page reloads (the hidden row is untouched by `EnsureTabs`).
- Cost: 2 extra round-trips per tracked doc per list load (one `INSERT`, one `SELECT`). Acceptable for a single-user personal tool with a small number of docs; batching is the natural next step if doc counts grow.

**Delete-vs-hide decision on the frontend:** The API returns only visible Tabs. The list page determines whether to "hide this tab" or "remove this doc" by counting how many visible CVs share the clicked row's `DocID`:

- Count > 1 → `POST /tracked-docs/{docId}/tabs/{tabId}/hide` (204)
- Count == 1 → `DELETE /tracked-docs/{docId}` (existing path; FK cascade removes tab rows)

No backend discriminator field is needed because the list is already visibility-filtered.

**Route shape:** `POST /tracked-docs/{docId}/tabs/{tabId}/hide` was chosen over `PATCH /tracked-docs/{docId}/tabs/{tabId}` with a `{"visible": false}` body. Rationale: the existing codebase uses action-oriented routes and there is no un-hide UI yet. If un-hide is added, migrate to `PATCH` with a `visible` field.

## Consequences

- `CONTEXT.md` updated: CV/Tab entries now mention the visibility flag; a "Hidden Tab" glossary entry is added.
- The `tracked_doc_tabs` migration runs after `tracked_docs` (timestamp `...0002 > ...0001`); FK `ON DELETE CASCADE` means removing a Tracked Doc also removes all its tab rows.
- `EnsureTabs` errors hard-fail `List` (treated as an infra fault, not a per-doc access problem).
- `providers.TabProvider` interface + `*db.DB` implementation (via sqlc) + slice-backed `MockTabProvider` follow the same patterns as `TrackedDocProvider`.
- Go identifiers use the short name `Tab` (not `TrackedDocTab`) — the DB table stays `tracked_doc_tabs`.
- `cvtemplates.NewService` takes a single `store` interface (embedding token-checker + doc + tab providers); `handlers.NewAuthHandler` likewise takes a single `authStore`.
