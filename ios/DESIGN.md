---
name: FastTrack iOS
description: A pocket notebook for triaging jobs and clearing what's due, in 1–3 minute bursts.
colors:
  brand-tint: "#6645d9"
  brand-tint-dark: "#7b5cf0"
  band-great: "#059669"
  band-good: "#10b981"
  band-fair: "#d97706"
  band-poor: "#dc2626"
  matched: "#059669"
  missing: "#dc2626"
  login-panel-top: "#2b2a46"
  login-panel-bottom: "#0c0a1d"
  login-ink: "#f5f4fc"
typography:
  display:
    fontFamily: "New York (SF Serif), Georgia, serif"
    fontSize: "28px"
    fontWeight: 400
  title:
    fontFamily: "New York (SF Serif), Georgia, serif"
    fontSize: "20px"
    fontWeight: 400
  headline:
    fontFamily: "SF Pro, system-ui, sans-serif"
    fontSize: "17px"
    fontWeight: 600
  body:
    fontFamily: "SF Pro, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 400
  meta:
    fontFamily: "SF Pro, system-ui, sans-serif"
    fontSize: "13px"
    fontWeight: 400
    fontFeature: "tnum"
  label:
    fontFamily: "SF Pro, system-ui, sans-serif"
    fontSize: "12px"
    fontWeight: 500
    fontFeature: "tnum"
rounded:
  chip: "5px"
  field: "12px"
  card: "14px"
spacing:
  row: "8px"
  card-gap: "10px"
  card-inset: "14px"
  gutter: "16px"
  group: "24px"
components:
  card:
    rounded: "{rounded.card}"
    padding: "16px"
  due-card-header:
    typography: "{typography.title}"
    padding: "{spacing.card-inset}"
  dot-label:
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "2px 6px"
  button-shortlist:
    backgroundColor: "{colors.brand-tint}"
    textColor: "#ffffff"
    size: "64px"
  button-dismiss:
    size: "64px"
  button-more:
    size: "48px"
  task-checkbox:
    rounded: "{rounded.chip}"
    size: "18px"
  login-submit:
    backgroundColor: "{colors.brand-tint}"
    textColor: "#ffffff"
    rounded: "{rounded.field}"
---

# FastTrack iOS — Design System Extension

> Extends: [`../DESIGN.md`](../DESIGN.md) (cited below as base §1–§9)
> Surface: ios (iPhone only, iOS 17+)
> Scope: [ADR 0035](../docs/adr/0035-ios-companion-is-a-triage-app.md). A screen it doesn't list needs a new decision.
> Source of truth: the prototype on `prototype/ios-triage` (`ios/FastTrack/Triage/Prototype/`).

This file records where the phone differs from the web system. Anything not covered here follows the base
doc. Where the two disagree on iOS, this file wins. The token frontmatter above is normative; the font sizes
in it are the Dynamic Type defaults at the Large size, so they're for reference only, never literals in views.

## Overview

**Creative North Star: "The Pocket Notebook"**

The web app is an operator's table. The phone is the notebook you carry: a white page, a few hairline-ruled
cards, names set in serif, and lines you tick off. Triage is one card at a time that you clear with your
thumb. Due is a page of notebook cards, one per job, with each to-do a ruled line and a checkbox.

The brand comes along as one violet tint, the Band and status palettes, monospaced digits and the voice.
Navigation, neutrals, SF type and appearance are left to iOS. The one loud moment is the login's brand panel,
a native copy of the web front door. Inside the app there is no gradient, no glass and no shadow.

Density follows Attio and Amie: tight, but not cramped. The test for every element is "does it change the
decision or move a to-do?" If it doesn't, it lives in job detail.

**Key Characteristics:**
- White `systemBackground` page, in light and dark alike.
- Cards are a 1px `separator` hairline at 14pt continuous corners. Nothing is lifted.
- Serif for entity names (company, today's date), SF for everything else, monospaced digits for data.
- Status is a dot-label: a coloured dot on a 10% tint, with the text in `.primary`.
- Secondary actions live in a native `Menu`, never in scattered inline buttons.
- Snappy 0.22s motion, and nothing swaps places mid-transition.

## Colors

One violet accent over system neutrals, with the Band palette carrying meaning.

### Primary
- **Brand Violet** (`brand-tint`, dark `brand-tint-dark`): the app-wide `.tint`. Used for the Shortlist
  button and swipe wash, a ticked checkbox, the login's Sign in button, and the "CV ready" dot. Defined once
  as `Color.brandTint` with light and dark values. Never written as a literal in views. The dark value exists
  because `#6645d9` on black is about 3.4:1 and fails as text.

### Secondary
- **Band palette** (`band-great`, `band-good`, `band-fair`, `band-poor`, from `frontend/src/lib/band.ts`):
  the words and meanings are unchanged from the web. On the Triage card and in job detail, a Band shows as a
  dot-label (the dot in the Band hue, the word in `.primary`), so Fair amber never has to pass 4.5:1 as text.
  The Due card header still colours the Band word; see Open.
- **Breakdown palette** (`matched`, `missing`, from `frontend/src/lib/scoreColour.ts`): only the breakdown
  row symbols use these.

### Tertiary
- **Login panel** (`login-panel-top` → `login-panel-bottom`, `login-ink`): the violet-black gradient, the
  aurora ribbons and the role cloud behind the login form. Converted from the web's oklch tokens. Used
  nowhere else.

### Neutral

| Web token (base §2) | iOS | Use |
|---|---|---|
| `--color-background` | `Color(.systemBackground)` | Every page, and every card fill |
| `--color-surface` | `Color(.systemBackground)` | Cards are the page colour; the hairline separates them |
| `--color-foreground` | `.primary` | Titles, dot-label text, row labels |
| `--color-muted` / `--color-faint` | `.secondary` | Meta lines, ages, counts. `tertiaryLabel` fails 4.5:1 and never carries text |
| `--color-border` | `Color(.separator)` | Card strokes, dividers, the outline of the round Dismiss and `···` buttons |
| — | `Color(.tertiaryLabel)` | Glyphs only: the unticked checkbox outline, chevrons |
| `--color-destructive*` | `Button(role: .destructive)` / `.red` | Exclude company, overdue, a failed draft |

Status hues on the Due list: overdue `.red`, due today `.orange`, CV ready Brand Violet, couldn't draft
`.red`.

**The One Violet Rule.** Violet means "this moves forward": Shortlist, done, ready. At most two violet moments
per screen. Dismiss is slate, never red, because it's undoable and red already means a gate or Poor.

**The Dot, Not the Word Rule.** Never colour a status word. Status is a 6pt dot on a 10% tint of its hue, with
the text in `.primary`. Orange or amber caption text fails 4.5:1.

**Appearance (base §9 exception).** The app follows the system appearance. Light is the reference design.
Dark is derived from it: semantic system colours plus the dark tint, with the same hairline cards on black.

## Typography

**Display font:** New York, the system serif (`.system(_, design: .serif)`). Nothing is bundled.
**Body font:** SF Pro through the system text styles.
**Data:** SF with `.monospacedDigit()`. This stands in for the web's JetBrains Mono rule without a second face.

**Character:** a serif masthead over a quiet sans. The serif names *who* (a company, a day). The sans says
*what* (a role, a Task, a status).

### Hierarchy
- **Display** (`.title`, serif): today's date on the Due list, and the company on the Triage card.
- **Title** (`.title3`, serif): the company on a Due card header.
- **Headline** (`.headline`, semibold): the role on the Triage card (up to 3 lines), and job titles in detail.
- **Body** (`.subheadline`): Task row titles and breakdown rows. `.body` is kept for the job description and
  notes.
- **Meta** (`.footnote`): the role line on a Due card, the Triage meta line ("London · Hybrid · £85,000–
  £100,000"), the Due summary line and the score. Monospaced digits wherever there are numbers.
- **Label** (`.caption`, medium): dot-label text, dates on Task rows. `.caption2` (11pt) is the floor.

**The Serif Is a Name Rule.** Serif is for entity names and the date masthead only. Never set a button, a
label, a number or a status in serif.

**The Text Style Rule.** No hard-coded point sizes anywhere, so Dynamic Type works. The one exception is the
login's decorative brand art (the role-cloud pills and the mark), which copies the web at fixed sizes.

Sentence case everywhere. Nothing is uppercased or tracked, except the login form's field labels, which
mirror the web front door.

## Layout

- **Gutter.** 16pt page margins (the system default). Don't override them.
- **Rhythm.** 24pt between groups (the header and the cards), 10pt between cards, 4–6pt between lines inside
  a card, 8pt vertical padding on a row.
- **Card insets.** 16pt on the Triage card, 14pt on a Due card header.
- **Alignment.** A Task row's checkbox lines up, by its visual edge, with the company name above it. Dividers
  under a card's header run the full card width. Dividers between rows are inset to the text (45pt on Task
  rows, 46pt on breakdown rows).
- **Triage page.** The card fills the space between the nav bar and the action bar. Up to three cards peek
  out underneath as hairline edges, each 6pt lower and 5% narrower. Below the card sits a 22pt gap, then the
  action bar.
- **Due page.** A `ScrollView` of cards (not a `List`): the serif date header, an optional horizontal "Coming
  up" strip of chases, then one card per job.

**The Card Is the Decision Rule.** A Triage card carries exactly what one keep-or-drop call needs. Every
field on it has to be able to change the decision. Description, the full breakdown, notes and contacts wait
for detail.

## Elevation & Depth

Flat. No surface casts a shadow in either appearance. Depth comes from the hairline and from stacking: the
Triage deck's peeking edges are the only "layers" in the app, and they're drawn with strokes, not shadows.
A dragged card doesn't gain a shadow; it tilts and takes on a 12% violet or grey wash.

The system's own materials (Liquid Glass bars, the tab bar, sheets, menus) are accepted as the platform's.
Custom views never use `.material`, `.glassEffect` or shadows.

**The Hairline Rule.** A surface is separated from the page by a 1px `separator` stroke and nothing else: in
light and dark, on cards, banners, chips and the round outline buttons.

## Shapes

Always `RoundedRectangle(cornerRadius:style: .continuous)`.

- **14pt** (`card`): Triage card, deck edges, Due cards, the undo banner.
- **12pt** (`field`): login fields and the Sign in button.
- **8pt**: the "Coming up" chase chips.
- **5pt** (`chip`): dot-labels and the 18pt task checkbox, a rounded square, not a circle.
- **Circles**: the action bar buttons. **Capsules**: the sync note and the prototype controls.
- System lists, sheets and menus keep their own radii.

## Components

### Triage card
*One calm card, read in a glance.*

```
┌──────────────────────────────────────┐
│ (● Great) 88 · 1 unknown          2d │  band row
│                                      │
│ Monzo                                │  company (serif .title)
│ Senior Backend Engineer              │  role (.headline, ≤3 lines)
│ London · Hybrid · £85,000–£100,000   │  meta (.footnote, secondary, mono digits)
│──────────────────────────────────────│  full-width divider
│ ⊘  Gate: Contract role               │  top 3 breakdown rows,
│──────────────────────────────────────│  inset dividers, ≥40pt each
│ ✓  Go                                │
│──────────────────────────────────────│
│ ✓  Series B–D                        │
│                                      │
│                  ⌄                   │  expand in place
└──────────────────────────────────────┘
       (✕)        (···)        (✓)         action bar
```

- **Band row.** A Band dot-label, the score in `.footnote` medium monospaced digits, then "· N unknown" in
  secondary when there are unknown rows. The age sits on the trailing edge in compact form ("2d", "3w").
  VoiceOver reads the row as one element.
- **Meta line.** Location, arrangement and salary joined with " · ". A missing value is left out, never shown
  as "—".
- **Breakdown rows.** The three rows that moved the score most. A gate is always first and reads "Gate: …".
  Each row is a symbol in a 20pt column, then the word in `.subheadline`. Unknown rows use secondary text.
  Rows aren't tappable (Corrections are out of scope).

  | Effect | Symbol | Colour |
  |---|---|---|
  | gated / blocked | `xmark.octagon.fill` | missing |
  | misses | `xmark.circle` | missing |
  | meets | `checkmark.circle` | matched |
  | favourite (Boost) | `star.circle` | matched |
  | level | `equal.circle` | secondary |
  | unknown | `questionmark.circle` | secondary |

- **Expand.** Tapping the card, or the `chevron.compact.down` at its foot, opens the card in place into a
  scrolling detail (meta, description, the grouped breakdown, note, contacts). The deck edges fade, and the
  action bar shrinks to 90% at 60% opacity. A 0.45s spring, or a 0.2s ease with Reduce Motion.
- **Accessibility sizes.** At `isAccessibilitySize` the card drops every breakdown row except a gate, and the
  role loses its line limit.

### Action bar
*Three round targets under the thumb.*

- **Dismiss**: a 64pt circle, `systemBackground` with a `separator` outline, and `xmark` in secondary. As the
  card drags left it fills slate and grows up to 15%.
- **`···`**: a 48pt circle, the same outline. It's a native `Menu` (below). It fades as a drag commits.
- **Shortlist**: a 64pt Brand Violet circle with a white `checkmark`. It grows up to 15% as the card drags
  right.
- 28pt between buttons. Each has a text accessibility label.

### Dot-label
*The one status primitive.* A 6pt dot in the hue, then `.caption` medium text in `.primary` (monospaced
digits), on a 10% tint of the hue with 5pt corners. Used for Bands, chase state ("● 3d overdue", "● Today"),
"● CV ready" and "● Couldn't draft".

### Due card (Notebook)
*One collapsible card per job.*

- **Header** (a button, 14pt padding, 6pt between its lines):
  - Line 1: the company in serif `.title3`, then Band (caption semibold, Band hue) and score (caption,
    secondary) 10pt after the name. On the trailing side, the to-do count and a chevron that rotates 90° when
    open.
  - Line 2: the role in `.footnote` secondary, with a dot-label for the most urgent item on the trailing side.
    It fades out while the card is open.
- **Body**: Task rows under a full-width divider. Only the first card starts open.
- Opening and closing use `.snappy(duration: 0.22)`. Rows fade in over 0.12s.

### Task row
*Attio and Amie style, one line to tick.*

- An 18pt rounded-square checkbox in a 44pt target. Unticked, it's a `tertiaryLabel` outline; ticked, a Brand
  Violet fill with a white check.
- The title in `.subheadline`, then a meta line: a dot-label plus the date in monospaced digits. Chases are the
  only items with dates; Tasks have none.
- Tapping the row (not the checkbox) opens a native `Menu`: Mark chased / Mark applied / Mark done, Snooze ▸,
  Log contact, View PDF, Retry, then a divider and Open job.
- Ticking strikes the title through and turns it secondary, plays a `.success` haptic, then removes the row
  0.5s later.
- A pending draft shows a small `ProgressView` in place of the checkbox. A failed draft shows a red-outlined
  retry square.

### Banners
- **Undo banner**: above the tab bar, `systemBackground` with a hairline at 14pt corners. It shows the action
  in `.footnote` semibold and the job in secondary, with **Draft CV** after a Shortlist and **Undo**. It lasts
  4 seconds, and a new action replaces it.
- **Sync note**: the same treatment as a capsule ("5 synced, 1 skipped (job removed)"), with no undo.
- **Offline row**: under the nav bar, `wifi.slash` "Offline · 3 queued" in `.footnote` medium secondary. It's
  a status, not an alert.

### Login
*The one brand-panel moment, a native copy of the web front door's mobile layout* (`frontend/src/components/auth/`).

- The upper panel: the violet-black gradient, slow drifting aurora ribbons (static with Reduce Motion), the
  compact role cloud and the FastTrack mark.
- A white sheet rises over it holding "Back on track" (`.title` bold), uppercase tracked caption labels,
  12pt fields with a hairline, and a Brand Violet **Sign in →**.
- There's no Sign up link (ADR 0035), and no film grain (it would need a bundled noise image).

## Do's and Don'ts

### Do:
- **Do** set company names and the date masthead in the system serif, and everything else in SF text styles.
- **Do** give every card a 1px `separator` stroke at 14pt continuous corners on a `systemBackground` fill.
- **Do** show every status as a dot-label with `.primary` text.
- **Do** put secondary actions in a native `Menu` opened from the row or the `···` button.
- **Do** animate toggles with `.snappy(duration: 0.22)`, and let content fade rather than swap.
- **Do** use `.monospacedDigit()` for scores, salaries, ages, counts and dates. Write dates in en-GB ("23 Oct").
- **Do** use the `CONTEXT.md` words: Triage queue, Shortlist, Dismiss, Band, Gate, Chase, Task.

### Don't:
- **Don't** add shadows, gradients, `.material` or `.glassEffect` to anything except the login panel.
- **Don't** use grouped grey backgrounds or tonal-step cards; the page and the card are both
  `systemBackground`.
- **Don't** colour status or Band text; Fair amber and overdue orange fail 4.5:1 at caption size.
- **Don't** scatter "···" or text buttons across rows. One menu per row or card.
- **Don't** hard-code point sizes outside the login's brand art.
- **Don't** add per-Task due badges. Tasks have no dates.
- **Don't** celebrate inbox zero: no illustration, no confetti.

---

## Gestures and feedback

| Input | Action | Colour | Haptic on commit |
|---|---|---|---|
| Drag right | **Shortlist** | Violet wash | `.success` |
| Drag left | **Dismiss** (Grade `no`) | Slate wash | `.impact(weight: .light)` |
| Swipe up, or **Skip** in the toolbar | Skip to the next card, which marks this one seen | — | none |
| Tap | Expand the card in place | — | — |
| `···` | Menu (below) | — | — |

**Drag.**
- **Direction lock.** After 10pt of travel, a drag that starts more vertical than horizontal is ignored.
- **While dragging.** The card follows the finger and rotates up to ±8° at full width (none with Reduce
  Motion). A violet or grey wash rises to 12% over the card, and the matching action button fills and grows.
  A `.selection` haptic fires when the commit threshold is crossed.
- **Commit.** At 35% of the card's width, or a flick predicted past 2.5× that. The card flies off in 0.25s
  ease-out (it fades with Reduce Motion). Below that, it springs back (`.spring(duration: 0.3, bounce: 0.15)`,
  no bounce with Reduce Motion).

**`···` menu**:
1. Shortlist and draft CV
2. Dismiss with reason… (a sheet listing the Grade reasons with checkmarks, plus a Dismiss button. A plain
   left swipe never shows it.)
3. Add note… (a medium-detent sheet with a `TextEditor`)
4. Exclude company (`role: .destructive`; no confirmation, because undo covers it)

**Draft CV.** A plain Shortlist doesn't draft. You start a draft from "Shortlist and draft CV" in the menu, the
**Draft CV** button on the Shortlist banner, or job detail.

**Undo.** Every card action shows the undo banner. The toolbar's `arrow.uturn.backward` button walks back
through the session's undo stack, Skip included, and is disabled when the stack is empty. Completing,
snoozing or marking applied on the Due list is undoable the same way.

**Accessible equivalents.**
- The action bar does exactly what the swipes do.
- **VoiceOver.** The card is one element. Its label reads: "Senior Backend Engineer, Monzo, London, Hybrid,
  £85,000–£100,000, 2 days old, Great, score 88". Its custom actions are Shortlist, Dismiss, Skip, and
  Details or Collapse details.
- Switch Control and Voice Control reach the same buttons and actions.

## Navigation

- **Tab bar** with two sections:
  - **Triage** (`rectangle.stack`)
  - **Due** (`checklist`), badged with the chases due plus ready drafts. Open Tasks aren't counted.
- **Triage.** An inline title, because a large title would take height from the card. Undo sits on the
  leading side and **Skip** on the trailing side.
- **Due.** An inline title "Due". The serif date header does the job of a large title.
- **Job detail.** Pushed from a Due card's **Open job**. It has the band row, the serif company header, then
  the detail body, with a `ShareLink` in the toolbar and the back swipe intact.
- **CV draft PDF.** **View PDF** opens the draft in `.quickLookPreview`, which brings sharing and Save to Files
  with no custom UI.

## States

**Queue clear.** `ContentUnavailableView` with `checkmark.circle`, "Queue clear" and "New jobs arrive after
the next scrape." No celebration.

**Loading.** One placeholder card shaped like the real one (`.redacted(reason: .placeholder)`). Never
"Loading…" text and never a bare spinner (base §7).

**Offline.**
- **Triage** keeps working from the cached cards, and the offline row counts the queued actions. When the
  cache runs out: "No more cached jobs" / "Reconnect to load more."
- **Sync.** After replay, the sync note shows "5 synced, 1 skipped (job removed)" for 4 seconds.
- **Due** is online only: "Due list needs a connection" with **Try again**. The tab badge hides while offline.
- **Job detail offline** reads from the cache, Add note queues, and Recruiter contacts shows "Available when
  online".

**Errors.** A failed action shows "Couldn't shortlist." with **Try again**, and the card goes back to its
place. A load failure is a `ContentUnavailableView` with **Try again**. An expired session goes back to login
and keeps the queued actions.

**CV draft states** (Due card rows and job detail):

| State | Leading | Meta |
|---|---|---|
| Pending | `ProgressView` | "Drafting…" in secondary |
| Ready | Checkbox (ticking it marks applied) | "● CV ready" plus the template name |
| Failed | Red retry square | "● Couldn't draft" |

## Motion

Every animation is functional, never decorative.

| Moment | Default | Reduce Motion |
|---|---|---|
| Drag | Follows the finger, rotates up to ±8° | No rotation |
| Below-threshold release | `.spring(duration: 0.3, bounce: 0.15)` | Same, no bounce |
| Commit | Flies off in the swipe direction (0.25s ease-out); the next card scales in from 94% | Fades |
| Expand card | `.spring(duration: 0.45, bounce: 0.12)` | 0.2s ease |
| Open a Due card, tick a row | `.snappy(duration: 0.22)`, rows fade over 0.12s | Same |
| Banner | Slides up from the tab bar | Fades |
| Login | The sheet rises in; the aurora drifts | Static |
| Push, sheets, menus | System | System |

Haptics fire whatever the Reduce Motion setting is, and respect the system's haptics switch through
`.sensoryFeedback`.

## Accessibility

This extends PRODUCT.md's AA baseline:

- **Contrast.** 4.5:1 for text, and 3:1 for large text and glyphs, in both appearances. Status text is
  `.primary` on a tint, so the hue only has to pass as a glyph.
- **Dynamic Type through AX5.** Text styles only, and the card has an accessibility-size layout.
- **Hit targets.** At least 44×44pt (the checkbox's target, not its 18pt square). Action bar buttons are
  48–64pt.
- **Not colour alone.** Every Band, breakdown row, status and swipe wash pairs its colour with a word or symbol.
- **Verification.** Simulator screenshots in light, dark and AX3, plus a VoiceOver walk through one triage
  burst on a device.

## Open

- The commit threshold and flick velocity still need measuring on a device.
- Whether 4 seconds is long enough to reach Undo while on the move.
- Whether the Triage card should hug its content instead of filling the page. The prototype fills it, which
  leaves white space under the three breakdown rows.
- The Due card header colours the Band word (caption semibold in the Band hue), which breaks The Dot, Not
  the Word Rule for Fair. Either make it a dot-label too, or accept it because the score sits beside it.
- Not yet verified: large Dynamic Type sizes across the app (the login was checked at AX3 and in dark).
