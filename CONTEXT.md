# Job Scraper

The domain language for Job Scraper — a personal job-hunting command centre that crawls job boards, tracks applications, and (newly) surfaces a user's CVs from Google Docs.

## Language

### Job search

**Job**:
A single listing scraped from a job board, shared as a DTO across crawl, enrich, and the frontend.
_Avoid_: Listing, posting, vacancy

**Source**:
A job board the scraper knows how to read, behind a per-board interface (e.g. `wis`, `greenhouse`).
_Avoid_: Provider, site, board

**Crawl** / **Enrich**:
The two decoupled scraper phases — Crawl discovers and enqueues job URLs; Enrich dequeues them and upserts full job details.

**Application**:
A user's tracked pursuit of a Job, moving through Statuses.
_Avoid_: Submission, app

**Status**:
A stage in the application pipeline (saved, applied, phone, interview, offer, rejected, plus custom). Each user has their own set.
_Avoid_: Stage, state, step

### CV templates

**Google Link**:
A user's stored, encrypted Google OAuth token granting `drive.readonly` access to their Drive.
_Avoid_: Connection, integration, Google auth

**Tracked Doc**:
A Google Doc a user has registered to pull CVs from. Persisted as a reference (`doc_id`, `added_at`), not its content.
_Avoid_: Document, file, source doc (the latter is only a UI column label)

**CV**:
A single **Tab** within a Tracked Doc, treated as one CV variant. Enumerated live from the Docs API; a per-tab **visibility** flag is persisted in `tracked_doc_tabs` so hidden tabs stay hidden across reloads.
_Avoid_: Resume, template, document. (The sidebar section is labelled "CV Templates", but a single item is a CV.)

**Tab**:
A native Google Docs tab. One Tracked Doc has one or more Tabs; each Tab is exactly one CV.

**Hidden Tab**:
A Tab whose persisted `visible` flag is `false`. It remains in the Google Doc and in `tracked_doc_tabs` but is filtered from the CV list. The visibility row is created automatically on the first `List` call for that doc (reconcile-on-list). Removing a hidden tab row from the DB is only needed when the whole Tracked Doc is removed (handled by FK cascade).
_Avoid_: Deleted tab, removed CV — the tab still exists in Google Docs.

## Relationships

- A **User** owns at most one **Google Link** and many **Tracked Docs**
- A **Tracked Doc** contains one or more **Tabs**; each **Tab** is exactly one **CV**
- A **Job** is pursued via at most one **Application** per user
- An **Application** has exactly one current **Status**

## Example dialogue

> **Dev:** "When a user adds a Tracked Doc with three tabs, do we store three CVs?"
> **Owner:** "No — we only store the Tracked Doc reference. The three CVs are read live from the doc's Tabs each time the list loads, so titles and content are never stale."
> **Dev:** "And a CV always belongs to one doc?"
> **Owner:** "Right. A CV is just a Tab. Remove the Tracked Doc and its CVs disappear from the list."

## Flagged ambiguities

- "CV template" (the user's phrase, kept as the sidebar section name) vs **CV** — resolved: the section is "CV Templates", but a single listed/viewed item is a **CV**, which is precisely one **Tab**.
- "doc" was used for both the Google Doc and a single CV — resolved: the whole file is a **Tracked Doc**; a single CV is a **Tab** within it.
