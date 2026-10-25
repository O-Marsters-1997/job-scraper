# FastTrack iOS — Design System Extension

> Extends: [`../DESIGN.md`](../DESIGN.md) (cited below as §1–§9)
> Surface: ios (iPhone only, iOS 17+)
> Scope: [ADR 0035](../docs/adr/0035-ios-companion-is-a-triage-app.md). A screen it doesn't list needs a new decision.

This file records only where the phone differs from the web system. Anything not covered here follows the
base doc. Where the two disagree on iOS, this file wins, and every exception is called out by section.

The phone is a triage companion for 1–3 minute bursts. The web app is an operator's table; this is a deck of
cards you clear with your thumb. The brand comes along as violet tint, the Band and status palette, mono
numerals and the voice. Navigation, neutrals, type and appearance are left to iOS.

---

## 1. Token mapping

### What iOS takes over

| Web token (§2) | iOS | Why |
|---|---|---|
| `--color-background` | `Color(.systemGroupedBackground)` | Adapts to dark mode and Increase Contrast; slate vs. system grey is not worth a custom colour |
| `--color-surface` | `Color(.secondarySystemGroupedBackground)` | Cards and grouped rows |
| `--color-surface-muted` | `Color(.tertiarySystemGroupedBackground)` | Inset blocks inside a card |
| `--color-foreground` | `.primary` (label) | |
| `--color-muted` | `.secondary` (secondaryLabel) | |
| `--color-faint` | `.secondary` | `tertiaryLabel` fails 4.5:1 and is never used for text someone has to read |
| `--color-border` / `-strong` | `Color(.separator)` / `Color(.opaqueSeparator)` | |
| `--color-destructive*` | `Button(role: .destructive)` / `.red` | Only Exclude company and Sign out |
| Sidebar indigo scale | Login screen background only | There is no sidebar; see §8 below |

### What stays brand

- **Tint.** App-wide `.tint`, the single colour for interactive elements (§2 "Violet brand accent").
  - Light: `oklch(0.55 0.18 285)` (`#6645d9`), the web `--color-primary`, unchanged.
  - Dark: `oklch(0.62 0.18 285)`. This is the only new colour value in this doc. It's needed because
    `#6645d9` on black is about 3.4:1 and fails as text. Against black it is about 5.8:1, and white labels on
    filled buttons are about 3.6:1, which passes AA large text (button labels are 17pt semibold). Verify both
    figures in the Simulator before shipping.
  - Defined once as a colour set in an asset catalogue with Any/Dark appearances, and never as a literal in
    views.
- **Band palette** (`frontend/src/lib/band.ts`): Great `#059669`, Good `#10b981`, Fair `#d97706`,
  Poor `#dc2626`. The meaning and labels are unchanged.
- **Breakdown palette** (`frontend/src/lib/scoreColour.ts`): matched `#059669`, missing `#dc2626`.
- **Status palette** (§2). Only Draft and Applied ever appear on the phone.
- **Tinted chips.** The web's `tintedChip` mixes towards white. On iOS:
  - Background: the hue at 14% opacity over the surface.
  - Light text: the hue mixed 65% towards black, as on web.
  - Dark text: the hue mixed 65% towards white.
  - Each chip has to hold 4.5:1 in both appearances. Check this with a contrast pass and don't assume it.

### Appearance (§9 exception)

§9's "no dark mode" is a web rule. On iOS the app **follows the system appearance**. Light is still the
reference design and gets designed first. Dark is derived from it: semantic system colours plus the dark tint
above. There's no hand-tuned indigo canvas, because dark mode is plain system black and greys with a violet
tint.

### Type (§3 exception)

The app uses San Francisco through the system text styles, so Dynamic Type works for free. Neither brand face
is bundled.

| Web (§3) | iOS text style | Use |
|---|---|---|
| `text-lg` page title | Nav title (`.largeTitle` on Due list, inline on Triage) | |
| `text-base` card title | `.title2` semibold on the Triage card, `.headline` in list rows | |
| `text-sm` body | `.body` | Description, notes |
| `text-data` | `.subheadline` | Company, location, meta rows |
| `text-xs` | `.footnote` | Chips, timestamps, banners |
| `text-2xs` | `.caption2` (11pt floor) | Never smaller |

The JetBrains Mono rule (§3, §9 "never replace mono for data") carries over like this:

- Score, salary, age and dates use `.monospacedDigit()` on the SF style, which keeps tabular figures without a
  second face.
- The source tag uses `.monospaced()` `.caption`.
- Section headers are sentence case, as in iOS grouped lists. Nothing is uppercased or tracked.

There are no hard-coded point sizes anywhere.

### Spacing and shape (§4)

- **Spacing.** The 4pt base step is kept. Screen gutters are the system margins (16pt); don't override them.
  Card padding is 20pt (web `p-5`), and gaps between card sections are 12–16pt.
- **Radius.** Always `RoundedRectangle(cornerRadius:style: .continuous)`.
  - The Triage card uses 16pt (`--radius-2xl`), because it's a sheet-sized surface.
  - Inset blocks and chips use 12pt and capsule respectively.
  - System lists and sheets keep their own radii.
- **Elevation.** As §4 says, there are no shadows on cards. In light mode the Triage card is a surface on the
  grouped background with a `separator` hairline. In dark mode the tonal step does that job and there's no
  stroke. A dragged card doesn't gain a shadow either; drag feedback is described in §4 below.

---

## 2. Density

"The table is the interface" (PRODUCT.md, principle 1) becomes **"the card is the decision"**. Each card
carries exactly what one keep-or-drop call needs and nothing for browsing. Density moves from the number of
rows on screen to how fast a decision gets made:

- One decision on screen at a time, legible at arm's length on a moving train.
- Every field that's on the card has to change the decision. Anything you only read once you're interested
  (description, full breakdown, note, contacts) lives in job detail.
- The score is evidence, not a headline. The Band leads, and the number sits beside it in the secondary colour
  (as with web `SuitabilityScoreValue`).
- The Due list is the one place the phone stays list-dense. It uses standard inset-grouped rows with two lines
  each.

---

## 3. Triage card anatomy

The queue is **vertical paging**: one card per page (`ScrollView` with `.scrollTargetBehavior(.paging)`,
iOS 17). The card fills the page inside the safe area, with the action bar under it.

```
┌──────────────────────────────────────┐
│ [Great] 82 · 1 unknown          2d   │  band row
│                                      │
│ Senior Backend Engineer              │  title (.title2 semibold, ≤3 lines)
│ Monzo                                │  company (.headline, secondary)
│ London · Hybrid                      │  location · arrangement
│ £85,000–£100,000                     │  salary (monospaced digits)
│ ──────────────────────────────────── │
│ ⊘  Gate: Contract role               │  top 3 breakdown rows
│ ✓  Go                                │
│ ✓  Series B–D                        │
│                                      │
│                      Details  ›      │  tap anywhere = push detail
└──────────────────────────────────────┘
     (✕ Dismiss)   (···)   (✓ Shortlist)    action bar
```

- **Band row.**
  - `BandChip` semantics in a tinted capsule (§1 above), then the score as `.subheadline` monospaced digits in
    secondary, then "· N unknown" when the breakdown has unknown rows. All of this matches
    `SuitabilityScoreValue`.
  - Age sits on the trailing edge in compact form ("2d", "3w") in monospaced digits.
  - An unscored job never reaches the queue (`scored=1`), so there's no empty state for this row.
- **Meta rows.** If a value is missing, the row is left out. A lone `—` with no label beside it means nothing
  on a card. This is the one place §8's em-dash rule doesn't apply.
- **Breakdown rows.**
  - Show the three rows that moved the score most. If a gate fired (`gated` or `blocked`), it's always first
    and starts with "Gate:".
  - The phone uses a vertical list instead of the web's chips, because chips wrap badly at large Dynamic Type
    sizes. Every row has an SF Symbol, a colour and a word, so colour never carries meaning alone:

    | Effect | Symbol | Colour |
    |---|---|---|
    | gated / blocked | `xmark.octagon.fill` | missing |
    | misses | `xmark.circle` | missing |
    | meets | `checkmark.circle` | matched |
    | favourite (Boost) | `star.circle` | matched |
    | level | `equal.circle` | secondary |
    | unknown | `questionmark.circle` | secondary |

  - Rows aren't tappable, because Corrections are out of scope.
- **Accessibility sizes.** At `dynamicTypeSize.isAccessibilitySize` the card drops the breakdown except a gate,
  and the title loses its line limit. The full breakdown is one tap away in detail. The card never scrolls
  internally, since that would fight the paging.

---

## 4. Gestures and feedback

| Input | Action | Colour | Haptic on commit |
|---|---|---|---|
| Drag right | **Shortlist** | Tint (violet) | `.success` |
| Drag left | **Dismiss** (Grade `no`) | Slate (`.secondary` fill) | `.impact(weight: .light)` |
| Swipe up | Skip to the next card, which marks this one seen | — | none (system paging) |
| Swipe down | Back to a skipped card within this session | — | none |
| Tap | Push job detail | — | — |
| `···` | Menu (below) | — | — |

**Why slate for Dismiss:** Dismiss can be undone and it isn't destructive. Red would also clash with the
Poor band and the gate rows on the same card.

**Drag.**
- **Direction lock.** After 10pt of travel, horizontal movement wins only if |dx| > |dy|. Otherwise the drag
  goes to the paging scroll.
- **While dragging.** The card follows the finger and rotates up to ±3°. On the side it's leaving, the action
  layer shows behind it: a tint or slate fill with the symbol and the word ("Shortlist" or "Dismiss"). At the
  commit threshold the label goes from 60% to full opacity, and a `.selection` haptic fires as the threshold is
  crossed in either direction.
- **Commit.** At 35% of the card's width, or a flick faster than about 800pt/s. Below that, the card springs
  back.

**`···` menu** (a `Menu`, which also opens on long-press of the card):
1. Shortlist and draft CV
2. Dismiss with reason… (a sheet with the Grade reasons from `GRADE_REASONS` as a checkmark list, plus a
   Dismiss button; never shown on a plain left swipe)
3. Add note… (a medium-detent sheet with a `TextEditor`)
4. Exclude company (`role: .destructive`; no confirmation, because undo covers it)

**Draft CV.** A plain Shortlist doesn't draft. You can start a draft in three places: "Shortlist and draft
CV" in the menu, the **Draft CV** button on the Shortlist confirmation banner, and the job detail actions.
Keeping the swipe a single decision keeps bursts fast.

**Undo.**
- **Banner.** Every action shows a banner above the tab bar, for example "Shortlisted · Senior Backend
  Engineer" with an **Undo** button, or **Draft CV · Undo** after a Shortlist.
  - Uses the `.regularMaterial` system material and lasts 4 seconds.
  - A new action replaces the current banner.
- **Undo button.** The toolbar's `arrow.uturn.backward` button walks back through this session's whole undo
  stack, one action at a time, and pages back to that card. It's disabled when the stack is empty.
- **Shake to undo.** Comes free through `UndoManager`.
- **Skip.** Skip has no undo; you get the card back by swiping down.

**Accessible equivalents.**
- **Action bar.** Under every card are three buttons, each at least 44pt: Dismiss (bordered, slate), `···`,
  and Shortlist (bordered prominent, tint, 56pt). Each one does exactly what its swipe does.
- **VoiceOver.**
  - The card is a single element. Its label reads: "Senior Backend Engineer, Monzo, London, hybrid, £85,000
    to £100,000, 2 days old, Great, score 82, gate: contract role".
  - Custom actions, in this order: Shortlist, Dismiss, Shortlist and draft CV, Dismiss with reason, Add note,
    Exclude company, Details.
  - After an action the app announces the result and the next card ("Shortlisted. Next: Platform Engineer,
    Cleo").
- **Switch Control and Voice Control.** These reach the same buttons and actions.

---

## 5. Navigation

- **Tab bar** with two sections (HIG: tabs are sections, never actions):
  - **Triage** (`rectangle.stack`)
  - **Due** (`checklist`), with a badge for chases due plus ready drafts. Open Tasks aren't counted, because
    they have no due date and would keep the badge lit all the time.
- **Triage.** Uses an inline title, a deliberate departure from HIG's large titles on top-level screens,
  because a large title would take height from a full-page card. The trailing toolbar holds the cards-left
  count ("24 left", monospaced digits) and Undo.
- **Due list.** Large title "Due", inset-grouped `List`.
- **Job detail.** Pushed from a card or a Due row, with an inline title and the left-edge back swipe intact.
  - The toolbar holds a `ShareLink` that shares the posting URL.
  - "Open original posting" is a row that opens Safari.
- **CV draft PDF.** Tap a ready-draft row to open the PDF (`GET /tailoring/drafts/{id}/pdf`) in
  `.quickLookPreview`. QuickLook brings sharing and "Save to Files" with no custom UI. **Mark applied** is a
  button on the row, not inside the preview, so the preview stays pure system UI.
- **Login.** A full-bleed sidebar indigo (`oklch(0.23 0.055 285)`) behind the FastTrack mark and a standard
  form. It's the one brand-panel moment and it's static, with no aurora.

**Due list sections**, shown in this order. Empty sections are hidden.

1. **Chases due.**
   - Each row shows the job title, the company, the chase date in monospaced digits and, when overdue,
     "overdue by N days" in `.red` (as in web `ChasesDueCard`).
   - Swipe actions: Done (leading), plus Snooze and Log contact (trailing). Snooze offers the same options as
     the web.
2. **CV drafts.** Each row shows the job title and the CV template name. The trailing control depends on the
   draft's state (§6).
3. **Open tasks.** Each row has a leading circle button that ticks the Task done, followed by the Task type and
   the job title. A ticked row strikes through, then leaves after one second and shows the undo banner.

Job detail sections, in order:

1. The header (the card's band row and meta rows).
2. The action bar: the same three buttons, plus Draft CV.
3. The description: sanitised HTML at `.body`, with link colour taken from the tint.
4. The full breakdown, grouped like the web `ScoreBreakdown` (Gates, Boosts, Met, Missed, Level, Unknown) as
   list sections, using the row symbols from §3.
5. Note.
6. Recruiter contacts, with a list and a **Log contact** sheet.
7. Open original posting.

---

## 6. States

**Loading.** One placeholder card shaped like the real one (`.redacted(reason: .placeholder)`). This follows
§7's skeleton rule: never "Loading…" text and never a bare spinner inside the shell. Pull to refresh on the Due
list is the system control.

**Queue clear (inbox zero).** `ContentUnavailableView`:
- Symbol `checkmark.circle`.
- Title "Queue clear".
- Description "New jobs arrive after the next scrape."
- No celebration and no illustration (§9). If the Due list has items, a **Go to Due** button switches tabs.

**Offline.**
- **Triage.** Keeps working from the cache of the next ~50 cards.
  - A slim inline row under the nav bar reads `wifi.slash` "Offline · 3 actions queued" in secondary text.
    It's a status, not an alert.
  - When the cache runs out: `ContentUnavailableView` "No more cached jobs" / "Reconnect to load more."
- **Sync note.** After replay, the banner shows "5 synced, 1 skipped" for 4 seconds. Here "skipped" means the
  job had been removed (a 404), and the banner says so: "1 skipped (job removed)". There's no undo on this
  banner.
- **Due list** (online only by ADR): `ContentUnavailableView` "Due list needs a connection" with a **Try
  again** button.
- **Job detail offline.** The description and breakdown come from the cache. Add note queues. The Recruiter
  contacts section shows "Available when online" in secondary text.

**Errors.**
- **Action failure online.** The banner reads "Couldn't shortlist." with **Try again**, and the card goes back
  to its place in the queue. Nothing fails silently.
- **Load failure.** `ContentUnavailableView` with **Try again**, the existing `LoadStateOverlay` pattern.
- **Expired session.** Back to login, with queued actions kept.

**CV draft states** (the CV drafts section and job detail):

| State | Row | Trailing |
|---|---|---|
| Pending | Title, then "Drafting…" in secondary text | `ProgressView`, no actions, row not tappable |
| Ready | Title, then the template name | **Mark applied** (bordered) |
| Failed | Title, then "Couldn't draft" in `.red` | **Try again** (re-posts the draft) |

Mark applied changes the Application's Status from Draft to Applied. The row leaves the list and the undo
banner appears.

---

## 7. Motion

This extends §7. Every animation stays functional, never decorative.

| Moment | Default | Reduce Motion |
|---|---|---|
| Drag | Follows the finger, rotates up to ±3° | Follows the finger, no rotation |
| Below-threshold release | `.spring(duration: 0.3, bounce: 0.15)` back to rest | Same spring, no bounce |
| Commit | The card leaves in the swipe direction, 0.25s ease-out, then the next card pages in | The card fades out over 0.2s and the next one fades in |
| Threshold reached | The label goes to full opacity | Same (opacity only) |
| Undo | The card returns with the reverse of how it left | Fade in |
| Banner | Slides up from the tab bar, 0.2s | Fades |
| Paging, push, sheets | System | System |

Haptics fire whatever the Reduce Motion setting is. They respect the system's haptics switch, because
`.sensoryFeedback` already does.

---

## 8. Platform conventions

**iOS wins** (the "ported from a website" test):

- System navigation (tab bar, stack, sheets, `Menu`) and the edge-swipe back gesture.
- System neutrals, SF type and following the system appearance (the §1 exceptions above).
- SF Symbols replace the Lucide-style SVGs (§6 "Icons"). The no-emoji rule still holds.
- Inset-grouped lists, system swipe actions on list rows, `ContentUnavailableView`, QuickLook and `ShareLink`.
- **Liquid Glass.**
  - When the app is built with the iOS 26 SDK, system bars, the tab bar and sheets take on Liquid Glass by
    themselves. That's accepted, because it's the platform's own material and not decoration.
  - Custom views never call `.glassEffect`. The deployment target is 17 and the Triage card is content, not
    a control.
  - The undo banner uses `.regularMaterial`.
  - This keeps PRODUCT.md's "no decorative glassmorphism" and §9 intact.

**The brand holds:**

- One violet tint, used for interactive elements and the Shortlist moment. That's at most two violet
  moments per screen, as §2 sets for the data area.
- The Band, breakdown and status palettes, with the same words and meanings as the web.
- Monospaced digits for data, and §8's voice: sentence case, short verbs, "en-GB" dates ("8 Jun 2026"), no
  superlatives.
- Borders and tone, never shadows, on content surfaces (§4).
- Rejected: gradients, gradient text, aurora inside the app, illustrations, celebration animation at inbox zero.

---

## 9. Accessibility

This extends PRODUCT.md's AA baseline:

- **Contrast.** 4.5:1 for text and 3:1 for large text and UI glyphs, in **both** appearances and with
  Increase Contrast on. That covers every tinted chip and the dark tint (§1). Measure; don't assume.
- **Dynamic Type through AX5.** No fixed sizes. The card has a layout for accessibility sizes (§3). Action
  bar labels may drop to icons only at AX sizes, but they keep their accessibility labels.
- **VoiceOver.** The card is one element with custom actions (§4). Banners are announced, and the Due badge
  count is read as "3 due".
- **Hit targets.** At least 44×44pt, with 8pt or more between neighbouring targets. The primary action is 56pt.
- **Not colour alone.** Every Band, breakdown row, swipe layer and overdue chase pairs its colour with a word
  or symbol.
- **Reduce Motion.** As set out in §7.
- **Verification.** Simulator screenshots in light, dark and AX3, plus a VoiceOver walkthrough of one triage
  burst on a device, because gestures and haptics need real hardware.

---

## Open for the prototype session

These are deliberately left to measure rather than guess:

- The commit threshold and flick velocity (35% and ~800pt/s are starting points).
- Whether the banner's 4 seconds is long enough to reach Undo while on the move.
- The exact card height split between the meta rows and the breakdown rows on the smallest supported iPhone.
