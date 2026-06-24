# FastTrack — Design System

> Category: Custom
> Surface: web

An automated personal job-hunting command centre. The backend crawls startup job
boards on a 6-hour cron, deduplicates and enriches listings, and persists them to a
database; the SolidJS + TanStack frontend surfaces the results and lets one user run
their whole application pipeline — browse, search, track, and update status — from a
single light-mode dashboard.

Do not invent values outside the palette and scale below.

---

## 1. Product Context & Visual Theme & Atmosphere

A focused operator's dashboard with a bold violet identity. The mood is **calm,
dense, and purposeful** — a quiet light canvas (`oklch(0.9842 0.0034 247.86)`)
holding white cards, framed on the left by a deep-indigo sidebar
(`oklch(0.23 0.055 285)`). One violet brand accent (`oklch(0.55 0.18 285)`)
carries every primary action and active state; everything else is slate-neutral.

The identity flows from the **login page**: a split-screen with a deep-indigo aurora
brand panel on the left and a clean white form on the right. Inside the app the same
violet system applies with restraint — aurora is permitted as faint decoration behind
shell chrome, empty states, and the dashboard header, but **never behind tabular
data**. The feeling is "admin console with personality": information per square inch
over vibes, with just enough violet to signal state and brand.

> **Tokens:** all colour tokens are OKLCH values in `frontend/src/styles.css`
> under `@theme`. The hex equivalents shown in §2 are for reference only.
> Chart.js (Canvas) cannot read CSS vars, so `frontend/src/lib/color.ts` exposes
> `cssVarHex(name)` which reads the live token and converts at call time — charts
> follow the active TweaksPanel preset automatically. No hardcoded hex in source.
>
> **Auth brand panel:** the login/signup illustration derives its gradient from
> `--color-sidebar` and its highlight/aurora/glow from `--color-primary` via
> `color-mix` — so the panel tracks the active preset. Source-tagged role pills
> keep their job-board brand hue (LinkedIn blue, Greenhouse green, etc.) but have
> lightness chosen for contrast: a dark panel uses light ink, a light custom panel
> switches to dark ink via `[data-auth-tone]`. Tweaks are applied globally in
> `main.tsx` before render so `/login` reflects the saved preset on cold load.

## 2. Color

Light mode only. Borders and surface tints — not shadows — do the structural work.

All tokens are OKLCH. Hex equivalents shown for visual reference.

**Canvas & surfaces**
- `--color-background` `oklch(0.9842 0.0034 247.86)` — app canvas (`#f8fafc`)
- `--color-surface` `oklch(1 0 0)` — cards, panels, inputs, menus (`#ffffff`)
- `--color-surface-muted` `oklch(0.9876 0.0017 247.84)` — table headers, hover rows, secondary chips (`#fafbfc`)

**Text**
- `--color-foreground` `oklch(0.2077 0.0398 265.75)` — primary text, headings, cell values (`#0f172a`)
- `--color-muted` `oklch(0.4455 0.0374 257.28)` — secondary text, company/location cells (`#475569`)
- `--color-faint` `oklch(0.5544 0.0407 256.79)` — metadata, placeholders, empty `—`, table heads (`#64748b`, ~4.76:1 on white for WCAG AA)

**Lines**
- `--color-border` `oklch(0.9288 0.0126 255.51)` — default hairline (`#e2e8f0`)
- `--color-border-strong` `oklch(0.869 0.0198 252.89)` — hover borders, breadcrumb divider (`#cbd5e1`)

**Violet brand accent** (used at most twice per screen in the main content area)
- `--color-primary` `oklch(0.55 0.18 285)` / hover `--color-primary-hover` `oklch(0.47 0.17 285)` (`#6645d9` / `#5535b8`) — white text on primary ~5.6:1, passes AA
- `--color-primary-foreground` `oklch(1 0 0)` (`#ffffff`)
- `--color-accent-subtle` `oklch(0.97 0.015 290)` · `--color-accent-border` `oklch(0.90 0.06 288)` · `--color-accent-text` `oklch(0.40 0.13 285)` — soft violet buttons, ghost hovers, active chips (~8.5:1 text on subtle bg)

**Deep-indigo sidebar panel** (its own self-contained scale)
- `--color-sidebar` `oklch(0.23 0.055 285)` · hover `--color-sidebar-hover` `oklch(0.29 0.06 285)` · border `--color-sidebar-border` `oklch(0.31 0.06 285)` (`#2a1f57` · `#372869`)
- foreground `--color-sidebar-foreground` `oklch(0.70 0.04 285)` → strong `--color-sidebar-foreground-strong` `oklch(0.95 0.02 290)` (idle nav ~5:1, strong ~11:1)
- active background `--color-sidebar-active` `oklch(0.62 0.20 300 / 0.16)` · active foreground `--color-sidebar-active-foreground` `oklch(0.84 0.13 295)` (~6:1 on active bg)
- badge surfaces: `--color-sidebar-badge` `oklch(1 0 0 / 0.07)` · `--color-sidebar-badge-active` `oklch(0.62 0.20 300 / 0.12)`

**Danger**
- `--color-destructive` `oklch(0.6368 0.2078 25.33)` / strong `--color-destructive-strong` `oklch(0.5771 0.2152 27.33)` / subtle `--color-destructive-subtle` `oklch(0.9705 0.0129 17.38)` (`#ef4444` / `#dc2626` / `#fef2f2`)

**Application-status palette** (chip defaults; tinted via `color-mix`)
- saved `oklch(0.5544 0.0407 257.42)` · applied `oklch(0.5461 0.2152 262.88)` · phone `oklch(0.5413 0.2466 293.01)` · interview `oklch(0.6658 0.1574 58.32)` · offer `oklch(0.596 0.1274 163.23)` · rejected `oklch(0.5771 0.2152 27.33)`
- Hex equivalents: saved `#64748b` · applied `#2563eb` · phone `#7c3aed` · interview `#d97706` · offer `#059669` · rejected `#dc2626`

> Status chips derive their background and text from the status's own hex at runtime:
> `background: color-mix(in srgb, <hex> 14%, white)`, `color: color-mix(in srgb, <hex> 78%, black)`.

## 3. Typography

- **UI / body / display**: **Plus Jakarta Sans** (`--font-sans`), weights 400/500/600/700. No separate display face — hierarchy comes from size and weight.
- **Data / numerics / IDs / source tags / dates**: **JetBrains Mono** (`--font-mono`), weights 400/500, with `tabular-nums`.
- Base body is **14px / 1.5** on the slate-900 foreground.

Scale:
| Utility | Size | Use |
|---|---|---|
| `text-2xs` | 10px | sidebar section labels (uppercase, tracked), badge labels, panel micro-text |
| `text-xs` | 12px | badges, table heads, scraped dates, field labels, tweaks panel captions |
| `text-data` | 13px | table cells, tweaks panel header titles |
| `text-sm` | 14px | body, buttons, nav, breadcrumb |
| `text-base` | 16px | card titles (semibold, snug) |
| `text-lg` | 18px | page titles — Jobs / Applications / Statuses (bold) |
| `text-xl` | 20px | decorative display glyphs in compact selectors |
| `text-auth-heading` | 1.85rem | login / signup form heading |
| `text-auth-subtext` | 0.95rem | login / signup subtitle |
| `text-auth-action` | 0.97rem | login / signup submit button |
| `text-display` | 2.55rem | auth brand panel headline (line-height bundled) |
| `text-display-prose` | 1.05rem | auth brand panel body prose (line-height bundled) |

All tokens are defined in `frontend/src/styles.css` under `@theme`. Do not use arbitrary `text-[Npx]` values — add a named token to `@theme` instead.

Weights: regular 400, medium 500, semibold 600, bold 700. Tracking: `tracking-tight`
(−0.025em) on the brand wordmark and page titles; `tracking-wide` on uppercase table
heads and the source badge; `tracking-wider` on sidebar section labels.

## 4. Spacing

4px base step. Common rhythm: nav links `px-2.5 py-1.5`, table cells `px-4 py-2.5`,
card padding `p-5` (header `pb-3`, footer `pb-5`), page gutter `px-7 py-6`, topbar
gutter `px-6`, gaps of `0.5rem`–`0.625rem`. Density is tight but breathable.

Fixed dimensions: controls `h-9` (36px) default, `h-8` (32px) small/icon; topbar and
sidebar header `h-14` (56px); sidebar `13.75rem` expanded / `3.5rem` collapsed. Auth
controls are deliberately softer: inputs `h-11 rounded-xl`, submit button `h-12 rounded-xl`.

Radius — defined as `--radius-*` tokens in `@theme`:
| Token | Value | Use |
|---|---|---|
| `--radius-sm` / `rounded-sm` | 0.25rem | — |
| `--radius-md` / `rounded-md` | 0.375rem | buttons, fields, nav links, icon buttons |
| `--radius-lg` / `rounded-lg` | 0.5rem | dropdown menus |
| `--radius-xl` / `rounded-xl` | 0.75rem | cards, table containers, list panels |
| `--radius-2xl` / `rounded-2xl` | 1rem | modals, auth cards |
| `rounded-full` | — | pills, avatars, status dots |

The TweaksPanel Sharp/Round control overrides `--radius-*` at runtime.

Elevation: cards, tables, and the shell rely on **borders only**. Shadows are
reserved for overlays — `shadow-xl` on dropdown content and the auth cards,
`shadow-2xl` on modals (over a `bg-black/30` + `backdrop-blur-[2px]` scrim).

## 5. Layout & Composition

App shell = **fixed deep-indigo sidebar + main column**. Sidebar (`oklch(0.23 0.055 285)`)
is collapsible (220px ⇆ 56px) with a FastTrack brand lockup at top, section groups
(`Main`, `Settings`), and nav links that use a violet-glow active pill
(`ring-1 ring-sidebar-active-foreground/20`). Main column is a 56px topbar
(breadcrumb left, circular avatar right) over a scrolling content region on the light
canvas. A faint violet aurora wash sits at the very top of the content region (inside
`.brand-aurora-shell`), fading before the data zone — restrained, never behind tables.

Primary surfaces: the **Jobs data table** (search + sortable columns + pagination),
the **Applications tracker**, and **Settings → Statuses**. Tables are full-width with
hairline row borders, a muted header band, uppercase faint heads, and hover-row
tinting. Slim 6px scrollbars.

## 6. Components

- **Button** (`rounded-md`, `h-9`, `text-sm` medium): `default` solid violet → violet-hover; `destructive` red; `outline` (border + surface, hover strengthens border/text); `secondary` "soft violet" (accent-subtle bg + accent-border + accent-text); `ghost`; `link`. Sizes `sm` / `default` / `lg` plus square icon variants; `focus-visible` violet ring with offset.
- **Card** (`rounded-xl`, border, no shadow): header `p-5 pb-3`, base-size semibold title, content `px-5`, footer `px-5 pb-5` pushed to bottom. Dashboard hero tiles carry `bg-accent-subtle/30 ring-1 ring-accent-border/40` for a restrained brand moment.
- **Badge** (`rounded-full`, `text-xs` medium): `default` solid violet, `secondary` muted, `outline`, and `source` — mono uppercase tracked slate chip for the job-board origin.
- **StatusBadge**: pill with a coloured dot + `color-mix` soft tint derived from the status hex (see §2).
- **Table**: wrapper scrolls x; `text-sm`; muted header band; `th` = `h-11` uppercase `text-xs` faint tracked; `td` = `px-4 py-4`, `whitespace-nowrap`; rows hairline-bordered, hover `surface-muted`, last row borderless. Text-heavy columns are width-capped (`max-w-[Npx] truncate`) with full text exposed via `title`.
- **Dropdown menu**: `rounded-lg` surface card, `min-w-[10rem]`, `shadow-xl`, animated in/out; items with leading 14px icons.
- **Field** (`.field` in `styles.css` / `Input` primitive): canonical style — bordered surface input, violet focus border + soft violet ring (`focus:ring-primary/10`); `Input` component references `.field` directly. Small muted medium label above (via `Label` cva).
- **Icons**: inline stroke SVGs (Lucide-style, `stroke-width:2`), 14–16px, `currentColor` — never emoji.

## 7. Motion & Interaction

`transition-colors` / `transition-all` on interactive elements; sidebar collapse is a
300ms `width` transition. Dropdown content animates with fade + 95% zoom on
open/close (`tailwindcss-animate`). Focus is always a visible violet ring
(`ring-2 ring-primary`, offset 1). Hover states: rows tint to `surface-muted`,
outline buttons strengthen border + text, ghost/icon buttons pick up `accent-subtle`.

**Brand aurora**: the auth brand panel uses animated ribbon blobs (`.auth-aurora`,
`authA1/2/3` keyframes). Inside the app a static, faint variant (`.brand-aurora-shell`
+ `.brand-ribbon`) appears at the top of the main content column and in empty/NotFound
states — opacity 0.04–0.07, `mix-blend-mode: multiply`, `pointer-events: none`.
Honour `prefers-reduced-motion: reduce` by disabling all aurora and rise animations
(already handled in `styles.css`).

## 8. Voice & Brand

Product name **FastTrack**. Wordmark = "Fast" + "Track" (Track coloured `var(--bright)`
on the auth panel, `text-primary` in light contexts). Mark = `FastTrackMark` SVG
(`components/brand-mark.tsx`), `currentColor`, used in Sidebar header and auth panel.

Copy is plain, operator-facing, sentence case: "Search by role or company…",
"Track application", "Edit application". Row actions are short verbs ("Apply ↗",
"Track application", "Edit", "Delete"). Column heads are short nouns. Dates render
`en-GB` short ("8 Jun 2026") in mono. Empty values are a faint em dash `—`, never
"N/A". No marketing superlatives or invented metrics.

## 9. Anti-patterns

- ❌ Dark mode, or any canvas other than `#f8fafc` / white surfaces.
- ❌ Colours outside this palette; more than ~two violet moments per screen (in the data area).
- ❌ Aurora/gradients **behind tabular data** — faint brand glow is only for shell chrome, empty states, and dashboard headers above the data zone.
- ❌ Gradient text (`background-clip: text`) — use a solid color.
- ❌ Glassmorphism as default — blurs and frosted cards are decorative noise. Aurora layers are `pointer-events-none` decoration, not card backgrounds.
- ❌ Drop shadows on cards — borders only. Shadows reserved for overlays.
- ❌ Hero images, decorative illustration, or emoji icons — inline stroke SVGs only.
- ❌ A second display typeface, or body text below 14px / table text below 12px.
- ❌ Generic "N/A" or invented stats; use a faint `—` for missing data.
- ❌ Replacing mono for data/IDs/dates with the sans face (lose `tabular-nums`).
- ❌ Sidebar in any colour other than the deep-indigo `oklch(0.23 0.055 285)` scale.
