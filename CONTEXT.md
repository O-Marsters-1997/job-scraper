# Job Scraper

The domain language for Job Scraper — a personal job-hunting command centre that crawls job boards, tracks applications, and (newly) surfaces a user's CVs from Google Docs.

## Language

### Job search

**Job**:
A single job opportunity, identified by a trusted ATS posting ID when one is available; the same Job may appear at several URLs.
_Avoid_: Listing, posting, vacancy

**Job URL**:
A source-specific link to a Job; several Job URLs may refer to the same Job when a trusted ATS posting ID establishes the match.
_Avoid_: Job identity, unique Job

**Job Candidate**:
A discovered job URL with cheap listing details, retained before its full Job details are fetched.
_Avoid_: Fully described Job, rejected Job

**Provisional Job**:
A fully described Job whose identity rests on a normalized Job URL until a trusted ATS posting ID establishes whether it matches another Job.
_Avoid_: Confirmed Job, duplicate Job

**Closed Job**:
A Job no longer treated as available because its source no longer advertises it after sufficient confirmation.
_Avoid_: Filled role, rejected application

**Source**:
A job-listing surface the scraper knows how to read behind a per-platform interface — an HTML job board (`wis`), an ATS platform adapter (`greenhouse`), or an Aggregator.
_Avoid_: Provider, site

**Board**:
A single company's listings on an ATS platform, identified by that platform's board token; a Company may have several Boards.
_Avoid_: Company page, account

**Verified Board**:
A Board successfully read from its ATS and associated with a Company through its careers site or explicit User confirmation.
_Avoid_: Detected Board, guessed Board

**Retired Board**:
A formerly verified Board no longer polled after its Company stops linking to it and two complete empty checks close its remaining Jobs.
_Avoid_: Closed Job, failed Board

**Company**:
A shared employer identity visible to all Users, which may be associated with zero or more verified ATS Boards.
_Avoid_: Employer, org, account

**Tracked Company**:
A Company a User has chosen to monitor, including Boards verified after the choice was made.
_Avoid_: Followed company, watched company, subscription

**Check Frequency**:
A User's requested interval for checking a Tracked Company's verified Boards, shared across that Company's Boards.
_Avoid_: Schedule, cron, polling interval

**Overdue Board**:
A Verified Board whose next check under its Check Frequency is more than an hour late, which signals that scheduling or checking has stalled.
_Avoid_: Stale Board, late Board, failed Board

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

**Scrape Run**:
One initiated discovery Source Target search or Verified Board check, complete when its Crawl work succeeds or has a recorded terminal failure.
_Avoid_: Source, schedule

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

**Option**:
One thing a User can pick a stance on — a technology, role, industry, level, work arrangement or
company stage — carrying the single atomic yes/no question Jev is asked about every Job on its
behalf. The full set (~160) is the bank, stored in `scoring_options` and shared by every User.
_Avoid_: Criterion, question — the bank replaces hand-written per-user criteria

**Dimension**:
One of six fixed groups an Option belongs to (`tech`, `role`, `domain`, `seniority`, `work`,
`stage`), fixed in Go code as `pair` (nice/avoid) or `multi` (nice only), deciding which stances a
Pick on that Option may take.
_Avoid_: Category, section

**Pick**:
A User's stance (nice, avoid, or block for `domain`) on one Option. Nice Picks in the same
Dimension are alternatives — a Job matching any one earns that Dimension's boost once; avoid Picks
cost a Job only when the Job has them.
_Avoid_: Preference (too broad), rule

**Answer**:
Jev's reply to one Option's question about one Job — three probabilities (yes/no/not_stated) plus
confidence, resolved to whichever is most likely (below 0.6, or not_stated, resolves unknown).
Shared across every User, keyed on the Job, its content fingerprint, the question's text hash and
the model, so identical questions across Users and Custom questions answer once.
_Avoid_: Score, judgement — an Answer is a fact about the Job, not a fit verdict

**Hard filter**:
A company-blocklist or no-go-tech check that runs in code before any Jev spend, at answer-effect
time rather than at ingest; a Job that trips every interested User's filter is never sent to Jev.
_Avoid_: Exclusion (too broad — Search Config's location exclusion is a Relevance-only filter, not
a Hard filter)

**Suitability**:
A 0–100 score per (Job, User) from a pure function over the User's Picks and the Job's cached
Answers — no per-Job Jev call, since Suitability is derived entirely from data already fetched
once. Gates notification and ranks the list, with one breakdown row per Pick explaining it.
_Avoid_: Relevance, fit score — keep distinct from Relevance

**Source Target**:
A User's chosen discovery search on a Source, identified by a search URL or criteria; tracking an ATS Company is a separate choice.
_Avoid_: Board config, tracked company, integration

**Role**:
A Source's purpose: `ats` Sources read verified Company Boards, while `discovery` Sources search for Job Candidates.
_Avoid_: kind (kind is value shape: board/url/filter), type

**Filter Source**:
A discovery Source searched through User-supplied keywords and structured filters rather than a fixed Board or search URL.
_Avoid_: keyword source, search source

**FilterField**:
A structured parameter declaration on a filter source — carries `Name` (the map key, e.g. `"region"`), `Label` (human-readable), and `Required`. `sourcespec` exposes declared fields via `LookupFilterFields(name)`; the `Create` handler validates submitted `filters` maps against them.
_Avoid_: filter param, filter key

**ScrapeRequest**:
A request to run a discovery Source Target now, including when it is first created or explicitly rerun. Discovery searches do not have a recurring schedule to cover a missed request.
_Avoid_: immediate scrape, manual scrape, trigger

**Search Config**:
A User's editable search criteria (role, location, keywords), Picks (in `preferences`), relevance
cutoff, and notify threshold — exactly one per User; the single source of truth feeding the
relevance gate, Suitability, and notifications.
_Avoid_: Settings, query — "preferences" is the Search Config field holding Picks, not a synonym
for the whole Search Config

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
- A **Job** has one or more **Job URLs**; matching trusted ATS posting IDs can establish that different URLs refer to the same **Job**
- A **Provisional Job** becomes part of an established **Job** when a trusted ATS posting ID confirms they are the same opportunity
- A **Job Candidate** may become a fully described **Provisional Job** or ATS-identified **Job** when a User's interest justifies fetching its details; a Candidate rejected by current filters remains available for later interest
- Every **Job Candidate** has a known **Source**; detail work without a Source is invalid
- A changed discovery **Source Target** or **Search Config** reconsiders retained **Job Candidates**, but only Candidates that pass the current cheap **Relevance** gate proceed to detail fetching
- A discovery **Source Target** runs once when added or explicitly rerun; recurring source checks are for **Verified Boards** of **Tracked Companies**
- A **Scrape Run** checks exactly one discovery **Source Target** or **Verified Board**; later Enrich failures are tracked separately and do not change its result
- A **Job** becomes a **Closed Job** after one successful, complete, nonempty **Board** check omits it; an empty **Board** needs two successful, complete checks before its formerly advertised Jobs close
- A non-ATS **Job** becomes a **Closed Job** only when its own page confirms unavailability; a transient fetch failure does not close it
- A **Job** may change while retaining its identity; changes to title, description, location, salary, or work arrangement make existing **Suitability** assessments stale for affected **Users**
- A **Job** is eligible for a new-Job alert only when first discovered and its **Suitability** reaches the User's notification threshold; later edits and reopening do not create another new-Job alert
- Promoting an older **Job Candidate** after changed User interest does not count as first discovery and does not send a new-Job alert
- A User who starts tracking a **Company** sees its already known open **Jobs** immediately and receives fresh **Suitability** assessments for them without new-Job alerts
- A changed **Search Config** never makes the User's existing **Suitability** assessments stale: saving Picks recomputes every scored Job for that User for free, from already-cached **Answers**, with no new Jev call
- An **Application** has exactly one current **Status**
- A **Source** iterates one or more **Boards** (ATS Sources only)
- A **User** defines zero or more discovery **Source Targets**; each Target maps to a supported **Source**
- A **Company** may have zero or more **Verified Boards**; a **User** makes one company-level choice to track all current and later Verified Boards of a **Tracked Company** at that User's **Check Frequency**
- An untracked **Company** may remain in the catalog without ATS inspection; tracking it starts Board discovery, and later verified Boards join the same tracking choice
- A superseded **Verified Board** remains polled until two complete empty checks confirm it has no open **Jobs**, then becomes a **Retired Board**
- When several **Users** track the same **Company**, its **Boards** are checked at the shortest requested **Check Frequency** with one shared check per Board
- A **Filter Source** Target carries a keyword `value` plus optional **FilterField** values in `filters`; a **ScrapeRequest** may be enqueued at creation time when `scrape_now: true`
- A **Job Candidate** can receive a cheap **Relevance** assessment per **User**; a fully described **Job** can receive a **Suitability** assessment per **User**. Neither is a property of the shared **Job**
- A **User** has exactly one **Search Config**

## Example dialogue

> **Dev:** "When a user adds a Tracked Doc with three tabs, do we store three CVs?"
> **Owner:** "No — we only store the Tracked Doc reference. The three CVs are read live from the doc's Tabs each time the list loads, so titles and content are never stale."
> **Dev:** "And a CV always belongs to one doc?"
> **Owner:** "Right. A CV is just a Tab. Remove the Tracked Doc and its CVs disappear from the list."
> **Dev:** "LinkedIn and Greenhouse link to the same ATS posting. Are those two Jobs?"
> **Owner:** "No. The trusted ATS posting ID makes them one Job with two Job URLs."
> **Dev:** "What if the LinkedIn result has no ATS ID yet?"
> **Owner:** "Keep it as a Provisional Job by its normalized Job URL, then merge it if a trusted ATS ID later proves the match."
> **Dev:** "If no current User wants a discovered listing, do we lose it?"
> **Owner:** "No. Keep a Job Candidate with its cheap details so later interest can justify the full fetch."
> **Dev:** "Can a Candidate's detail work proceed if its Source is missing?"
> **Owner:** "No. Reject it with an error; our scrapers handle known Sources."
> **Dev:** "If I change my search, do all stored Candidates get full details?"
> **Owner:** "No. Recheck them soon, but only fetch details for Candidates that pass the new cheap gate."
> **Dev:** "Does the same Job stay frozen if the company edits its salary?"
> **Owner:** "No. Update the Job from the trusted source and reassess Suitability when its title, description, location, salary, or work arrangement changes."
> **Dev:** "If I change my Picks, do all my old Jobs get rescored immediately?"
> **Owner:** "Yes, for free. Every Answer is already cached, so saving just re-runs the formula over every Job I've been scored for — no new Jev call."
> **Dev:** "Does reopening an old Job announce it as new again?"
> **Owner:** "No. The new-Job alert belongs to first discovery only."
> **Dev:** "Does every new Job produce an alert before Suitability is known?"
> **Owner:** "No. Alert only after my Suitability score reaches my notification threshold."
> **Dev:** "If I start tracking a Company later, do its existing open Jobs wait for another Board check?"
> **Owner:** "No. Show them immediately, assess their Suitability for me, and do not alert me as if they were new."
> **Dev:** "Does one empty Board check mean the company filled every role?"
> **Owner:** "No. Two complete, successful empty checks can close those Jobs as no longer advertised; we do not know whether the roles were filled."
> **Dev:** "What if the Board still lists other Jobs but drops this one?"
> **Owner:** "One complete successful check is enough to close that missing Job."
> **Dev:** "If a Company has separate regional ATS Boards, do I track each one?"
> **Owner:** "No. I track the Company once, and that covers all of its verified Boards."
> **Dev:** "Is an ATS-looking URL enough to attach a Board to the Company?"
> **Owner:** "No. Read the Board successfully and confirm the Company link through its careers site or my explicit confirmation."
> **Dev:** "Can I track a Company before we know which ATS it uses?"
> **Owner:** "Yes. My tracking choice remains active, and a Board starts contributing Jobs when it is later verified."
> **Dev:** "Would I set a different Check Frequency for each regional Board?"
> **Owner:** "No. I set one Check Frequency for the Tracked Company, and it applies to all its Boards."
> **Dev:** "Another User wants that Company checked more often. Do we fetch its Boards twice?"
> **Owner:** "No. Share one Board check at the shortest requested Check Frequency."
> **Dev:** "If I paste a direct ATS Board URL, does it become a standalone Source Target?"
> **Owner:** "No. Identify or create its Company, attach the verified Board, and track the Company."
> **Dev:** "If a detail fetch is repeated after a failure, does that create a second Job?"
> **Owner:** "No. Repeated fetching is acceptable; the same opportunity keeps one canonical Job identity."
> **Dev:** "If a detail cannot be completed, does that change the Scrape Run result?"
> **Owner:** "No. The run reports discovery; track the later detail failure separately."

## Flagged ambiguities

- "CV template" (the user's phrase, kept as the sidebar section name) vs **CV** — resolved: the section is "CV Templates", but a single listed/viewed item is a **CV**, which is precisely one **Tab**.
- "doc" was used for both the Google Doc and a single CV — resolved: the whole file is a **Tracked Doc**; a single CV is a **Tab** within it.
- "board" meant both a **Source** and the per-company ATS unit — resolved: a **Source** is a platform/site adapter; a **Board** is one company's listings on an ATS (a `{board_token}`) that a Source iterates.
- "relevance" vs "suitability" — resolved: **Relevance** is the cheap pre-persistence heuristic gate signal; **Suitability** is the post-persistence LLM fit score. Both are 0–100 and per **User**, but differ in input (card vs full text), cost (free vs LLM), and timing.
- "score on a Job" read as a property of the shared **Job** — resolved: a score is per-**User** (a **Job**↔**User** assessment, modelled like **Application**), never a column on the shared catalog.
- "source target value" for WIS was ambiguous — resolved: for **Filter Source** targets the `value` column is the keyword string (what to search for); additional structured parameters (e.g. region) live in the `filters` JSONB column, not in `value`.
- "Job" previously meant one scraped URL — resolved: trusted ATS posting identity defines one **Job**; URLs are **Job URLs** and can be aliases.
- "Job without an ATS ID" — resolved: keep a **Provisional Job** by normalized **Job URL** and merge it only after a trusted ATS ID confirms identity.
- "partial Job" — resolved: discovery metadata before detail fetching is a **Job Candidate**; a fully described Job without a trusted ATS ID is a **Provisional Job**.
- "scrape once" implied a Job never changes — resolved: a **Job** keeps its identity across edits, while relevant changed details can require new **Suitability** assessments.
- "deliver once" could mean one fetch, one ingest request, or one Job — resolved: repeat fetches and ingest requests are acceptable, while the same opportunity keeps one canonical **Job** identity.
- "unclassified detail" suggested a general crawler — resolved: every detail task must retain its **Source**, and missing Source identity is an error.
- "stale score" previously had two causes with different responses — resolved: changed Job details
  change the content fingerprint and re-queue an answer effect (Answers are re-asked); a changed
  User Search Config has no stale-score response at all, because Save recomputes every scored Job
  immediately from cached Answers.
- "closed Job" could imply the position was filled — resolved: **Closed Job** means the source no longer advertises it after sufficient confirmation; one complete nonempty check can close a missing Job, while an empty Board requires two complete successful checks.
- "Tracked Company" previously meant one board-specific **Source Target** — resolved: it is one User choice covering current and later verified **Boards**, even when no Board is yet known.
- "Check Frequency" previously belonged to each ATS **Source Target** — resolved: it belongs to the User's **Tracked Company** and applies across its verified **Boards**.
- "Source Target" previously included ATS Board tracking — resolved: **Source Target** is a discovery search; direct ATS Board entry resolves to a **Company** and its **Tracked Company** choice.
- "scheduled scraping" previously included recurring discovery searches and HTML detail refresh — resolved: discovery runs once or on explicit rerun; only matched ATS Boards of Tracked Companies are checked automatically on a recurring schedule.
- "verified Board" previously meant a detected ATS-looking URL — resolved: a **Verified Board** also requires a successful read and Company association evidence.
