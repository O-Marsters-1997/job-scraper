# Job Scraper

The domain language for Job Scraper — a personal job-hunting command centre that crawls job boards, tracks applications, and (newly) surfaces a user's CVs from Google Docs.

## Language

### Job search

**Job**:
A single listing scraped from a job board, shared as a DTO across crawl, enrich, and the frontend.
_Avoid_: Listing, posting, vacancy

**Source**:
A job-listing surface the scraper knows how to read behind a per-platform interface — an HTML job board (`wis`), an ATS platform adapter (`greenhouse`), or an Aggregator.
_Avoid_: Provider, site

**Board**:
A single company's listings on an ATS platform, identified by a board token (e.g. a Greenhouse `{board_token}`). One Source iterates many configured Boards.
_Avoid_: Company page, account

**ATS**:
An applicant tracking system (Greenhouse, Lever, Ashby, Workable, Recruitee, Personio) exposing a public, unauthenticated jobs API — the Tier-1 source of truth, extracted via API not HTML.
_Avoid_: Platform (when ambiguous), provider

**Aggregator**:
A discovery-only Source (LinkedIn, Indeed) used to find job URLs, never to extract content; a discovered URL is re-classified and extracted from its underlying ATS where possible.
_Avoid_: Board — an Aggregator is not a Source of content

**ATSType**:
The classification `Detect(url)` assigns a discovered URL before dispatch — a known ATS platform, an Aggregator, or unknown-HTML — routing it to the cheapest viable extraction method.
_Avoid_: Provider type, kind

**Crawl** / **Enrich**:
The two decoupled scraper phases — Crawl discovers and enqueues job URLs; Enrich dequeues them and upserts full job details. ATS API Sources complete in Crawl alone (the list call returns full details), so they have no Enrich phase.

**Application**:
A user's tracked pursuit of a Job, moving through Statuses.
_Avoid_: Submission, app

**Status**:
A stage in the application pipeline (saved, applied, phone, interview, offer, rejected, plus custom). Each user has their own set.
_Avoid_: Stage, state, step

### Scoring & criteria

**Relevance**:
A 0–100 heuristic score of a Job's listing-card signals (title/company/location) against a User's criteria, computed pre-persistence; the relevance cutoff gates whether the Job advances to the next expensive stage.
_Avoid_: Match score, filter score — keep distinct from Suitability

**Suitability**:
A 0–100 LLM (Claude Haiku) score of how well a Job fits a User's criteria, computed from full job text against a rubric after persistence; gates notification and ranks the list.
_Avoid_: Relevance, fit score — keep distinct from Relevance

**Search Config**:
A User's editable search criteria (role, location, keywords), suitability rubric, relevance cutoff, and notify threshold — exactly one per User; the single source of truth feeding the relevance gate, the suitability scorer, and notifications.
_Avoid_: Settings, preferences, query

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
- A **Source** iterates one or more **Boards** (ATS Sources only)
- A **Job** carries a **Relevance** and **Suitability** score per **User** — a per-user assessment, sibling to **Application**, not a property of the shared **Job**
- A **User** has exactly one **Search Config**

## Example dialogue

> **Dev:** "When a user adds a Tracked Doc with three tabs, do we store three CVs?"
> **Owner:** "No — we only store the Tracked Doc reference. The three CVs are read live from the doc's Tabs each time the list loads, so titles and content are never stale."
> **Dev:** "And a CV always belongs to one doc?"
> **Owner:** "Right. A CV is just a Tab. Remove the Tracked Doc and its CVs disappear from the list."

## Flagged ambiguities

- "CV template" (the user's phrase, kept as the sidebar section name) vs **CV** — resolved: the section is "CV Templates", but a single listed/viewed item is a **CV**, which is precisely one **Tab**.
- "doc" was used for both the Google Doc and a single CV — resolved: the whole file is a **Tracked Doc**; a single CV is a **Tab** within it.
- "board" meant both a **Source** and the per-company ATS unit — resolved: a **Source** is a platform/site adapter; a **Board** is one company's listings on an ATS (a `{board_token}`) that a Source iterates.
- "relevance" vs "suitability" — resolved: **Relevance** is the cheap pre-persistence heuristic gate signal; **Suitability** is the post-persistence LLM fit score. Both are 0–100 and per **User**, but differ in input (card vs full text), cost (free vs LLM), and timing.
- "score on a Job" read as a property of the shared **Job** — resolved: a score is per-**User** (a **Job**↔**User** assessment, modelled like **Application**), never a column on the shared catalog.
