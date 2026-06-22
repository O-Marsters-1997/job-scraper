---
target: my entire main frontend pages
total_score: 23
p0_count: 1
p1_count: 2
timestamp: 2026-06-22T16-31-47Z
slug: frontend-src-routes-auth
---
# Critique: main frontend pages (`frontend/src/routes/_auth`)

Surfaces reviewed (live, mock mode, desktop 1440 + mobile 390): Overview, Jobs, Applications, Insights, CV Templates, Settings → Statuses / Searches / Integrations.

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 2 | Raw "Loading…" text instead of skeletons; thrown errors crash to a dev fallback |
| 2 | Match System / Real World | 3 | Clean operator language (Saved/Applied/Phone Screen); solid |
| 3 | User Control and Freedom | 3 | Search, sort, pagination, edit/delete all present |
| 4 | Consistency and Standards | 2 | Jobs uses a kebab row-menu; Applications uses inline text links for the same actions |
| 5 | Error Prevention | 2 | Destructive "Delete" is a plain link with no confirmation |
| 6 | Recognition Rather Than Recall | 3 | Nav is icon + label; good discoverability |
| 7 | Flexibility and Efficiency | 2 | No keyboard shortcuts or bulk actions on a power-user tool |
| 8 | Aesthetic and Minimalist Design | 3 | Restrained, on-brand; four identical stat tiles are the one cliché |
| 9 | Error Recovery | 1 | Unstyled TanStack default error boundary ships to users |
| 10 | Help and Documentation | 2 | Inline hints exist; no contextual help (acceptable for a personal tool) |
| **Total** | | **23/40** | **Acceptable — solid bones, broken edges** |

## Anti-Patterns Verdict

**LLM assessment:** This does NOT read as AI slop. The restraint is real and earned — single teal accent, hairline borders doing the structural work, mono `tabular-nums` for data, near-black sidebar. It looks like a tool someone uses, which is exactly the brand. The one tell is the Overview's four identical stat cards (Jobs in database / New today / Active applications / Response rate): equal-weight metric tiles in a row is the SaaS-dashboard reflex.

**Deterministic scan:** `detect.mjs` over `src/routes` returned `[]` (clean). No gradient text, eyebrows, side-stripes, or glassmorphism. The detector does not parse Solid JSX semantics, so it is a floor, not a ceiling.

**Visual evidence:** 10 full-page screenshots captured in mock mode. The damning ones: Insights and CV Templates both render as a raw black-on-white "Something went wrong! / Hide Error" screen with mono error text.

## Overall Impression

The happy path is genuinely good — Overview, Jobs, Applications, and Statuses are calm, dense, legible, and on-brand. The problem is the *edges*: two of seven nav destinations currently render as a broken dev error screen, the breadcrumb is wrong on half the pages, loading shows raw "Loading…" text, and the whole thing is unusable on mobile. The bones are a 32; the unfinished edges drag it to a 23. Single biggest opportunity: **make the non-happy-path states (error, loading, mobile) as finished as the happy path.**

## What's Working

1. **Disciplined visual system.** Borders-not-shadows, one teal accent used sparingly, slate-neutral everything else. Statuses and Integrations are textbook restrained-product UI.
2. **Data typography.** Mono + `tabular-nums` for dates, scores, and counts reads as precise and scannable — the right call for a read-heavy table tool.
3. **Status vocabulary.** The `color-mix` status chips (Saved/Applied/Phone Screen/Interview/Offer/Rejected) are consistent across Overview, Applications, and Statuses, and carry state without shouting.

## Priority Issues

### [P0] The global error boundary is TanStack Router's raw dev fallback
`__root.tsx` registers no `errorComponent`, so any thrown error renders the built-in black-title + red-bordered mono "Something went wrong! / Hide Error" screen — no sidebar, no branding, no recovery path. Insights crashes here (`Cannot read properties of null (reading 'getComputedStyle')`, from chart.js measuring a canvas before mount) and CV Templates crashes here (`Failed to fetch`). Even where the underlying cause is environment-specific, **the fallback itself ships to production and looks broken.**
- **Why it matters:** Two of seven main nav pages currently look like the app crashed. For a tool whose brand is "respects your attention / quiet confidence," a dev stack-trace screen is the opposite signal.
- **Fix:** Register a branded `errorComponent` (and `defaultErrorComponent`) that renders inside the app shell — heading, plain-language message, "Try again" / "Back to Jobs" actions, using `--color-destructive-subtle`. Add route-level boundaries so one chart failing doesn't blank the page. Separately, fix the Insights chart `getComputedStyle` null (guard the canvas ref / mount the chart in `onMount`).
- **Suggested command:** `/impeccable harden`

### [P1] Mobile is unusable — the sidebar never collapses
At 390px the 220px dark sidebar stays fixed, leaving ~170px for content. The Jobs table is crushed to the TITLE column alone; the breadcrumb wraps to two lines. The collapse affordance exists but is manual-only.
- **Why it matters:** The whole content area is a sliver. `DESIGN.md` and the product register both call for *structural* responsive behavior (collapse the sidebar), and it isn't happening.
- **Fix:** Auto-collapse to the 56px icon rail below ~768px, or move to an off-canvas drawer with a hamburger in the topbar. Let the table scroll within its own container, not the viewport.
- **Suggested command:** `/impeccable adapt`

### [P1] Breadcrumb reads "Job Scraper / Job Scraper" on four pages
`Topbar.tsx:8` hardcodes `PAGE_LABELS` for only `/overview`, `/jobs`, `/applications`, `/settings/statuses`. Insights, Searches, Integrations, and CV Templates fall through to the `"Job Scraper"` fallback, so the breadcrumb duplicates the product name and stops telling the user where they are.
- **Why it matters:** Breadcrumbs are the one persistent "you are here" cue; on half the pages it's broken.
- **Fix:** Derive the label from the route (route `staticData`/`context` title, or a generated map) instead of a hand-maintained dictionary, so new routes can't regress.
- **Suggested command:** `/impeccable harden` (or a direct one-line fix)

### [P2] Loading is raw "Loading…" text, not skeletons
Searches renders the page header then literal "Loading…" body text. The product register explicitly mandates skeleton states over mid-content text/spinners.
- **Why it matters:** Layout jumps when data arrives; reads as unfinished against the rest of the app.
- **Fix:** Skeleton rows that match the loaded list/table shape.
- **Suggested command:** `/impeccable harden`

### [P2] Overview's four identical stat cards are the dashboard cliché
Equal-sized metric tiles in a 4-up row is the one element a stranger would call "generic SaaS." It also flattens hierarchy: "Response rate" (an outcome) gets the same weight as "Jobs in database" (a static count).
- **Why it matters:** Misses a chance to lead with what the operator actually acts on, and it's the page's only AI tell.
- **Fix:** Break the symmetry — promote the one or two metrics that drive action (active applications / response rate), demote the rest to a compact inline strip. Vary size/treatment so the row has a focal point.
- **Suggested command:** `/impeccable layout`

## Persona Red Flags

**Alex (Power User):** No keyboard shortcuts for search/navigation/row actions. No bulk selection on Jobs or Applications — moving five jobs through the pipeline is five separate row menus. Sortable columns and pagination are the only accelerators.

**Sam (Accessibility):** Faint metadata (`--color-faint` `#94a3b8`) on white is ~2.8:1 — below 4.5:1 for the "Scraped" timestamps, table sub-rows, and helper copy. The Searches/Statuses helper text and the "5 of 12 responded" sublabels are at risk. Color-only status dots need a text label backup (they have one — good). Error screens give a screen reader a bare stack-trace string.

**Riley (Stress Tester):** Found it immediately — Insights and CV Templates throw to the raw boundary. Delete on Applications has no confirmation step, so a misclick is destructive and unrecoverable.

## Minor Observations

- Em dash in the Statuses subtitle ("Custom stages for your pipeline — assign any colour") — the design voice bans em dashes; use a period or parentheses.
- En dash in the Searches subtitle ("each runs on its own 6–hour cycle") — should be a hyphen ("6-hour").
- Applications uses inline "Edit / Delete" text links while Jobs uses a kebab row-menu for the same class of action — pick one row-action vocabulary.
- The TanStack Router devtools badge is visible in every capture — confirm it's stripped from production builds.

## Questions to Consider

- The happy path is a 32 and the edges are a 10. What if "done" for a page meant its error, loading, and empty states shipped at the same finish as its loaded state?
- Is mobile actually in scope? If this is desktop-only by design, say so and lock a min-width; if not, the sidebar is the first thing to fix.
- Delete is one mistaken click from gone. Is an undo toast more in keeping with "quiet confidence" than a confirm dialog?
