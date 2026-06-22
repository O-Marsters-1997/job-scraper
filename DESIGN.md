# Job Scraper — Design System

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

A focused operator's dashboard, not a marketing site. The mood is **calm, dense,
and trustworthy** — a quiet light canvas (`oklch(0.9842 0.0034 247.86)`) holding
white cards, framed on the left by a single near-black sidebar
(`oklch(0.2094 0.0199 248.8)`). One teal brand accent (`oklch(0.6274 0.1078 185.99)`)
carries every primary action and active state; everything else is slate-neutral. The
feeling is "admin console you live in all day": shadcn / Tabler / TailAdmin light-mode
lineage — information per square inch over vibes, with just enough colour to signal
state. No gradients, no hero imagery, no decoration for its own sake.

> **Tokens:** all colour tokens are defined as OKLCH values in `frontend/src/styles.css`
> under `@theme`. The hex equivalents shown in §2 below are for reference only.

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

**Teal brand accent** (used at most twice per screen)
- `--color-primary` `oklch(0.6274 0.1078 185.99)` / hover `--color-primary-hover` `oklch(0.5254 0.0902 185.8)` (`#0f9d92` / `#0a7b72`)
- `--color-primary-foreground` `oklch(1 0 0)` (`#ffffff`)
- `--color-accent-subtle` `oklch(0.9836 0.0142 180.72)` · `--color-accent-border` `oklch(0.91 0.0927 180.43)` · `--color-accent-text` `oklch(0.3861 0.059 188.42)` — secondary/"soft teal" buttons, avatar chip, ghost hovers (`#f0fdfa` · `#99f6e4` · `#134e4a`)

**Dark sidebar panel** (its own self-contained scale)
- `--color-sidebar` `oklch(0.2094 0.0199 248.8)` · hover `--color-sidebar-hover` `oklch(0.2783 0.033 247.38)` · border `--color-sidebar-border` `oklch(0.2783 0.033 247.38)` (`#111921` · `#1b2a38`)
- foreground `--color-sidebar-foreground` `oklch(0.6417 0.0422 250.84)` → strong `--color-sidebar-foreground-strong` `oklch(0.9243 0.0169 236.7)` (`#7a8fa6` → `#dce8f0`)
- active background `--color-sidebar-active` `oklch(0.7038 0.123 182.5 / 0.13)` · active foreground `--color-sidebar-active-foreground` `oklch(0.8549 0.1251 181.07)` (`rgba(20,184,166,0.13)` · `#5eead4`)

**Danger**
- `--color-destructive` `oklch(0.6368 0.2078 25.33)` / strong `--color-destructive-strong` `oklch(0.5771 0.2152 27.33)` / subtle `--color-destructive-subtle` `oklch(0.9705 0.0129 17.38)` (`#ef4444` / `#dc2626` / `#fef2f2`)

**Application-status palette** (chip defaults; tinted via `color-mix`)
- saved `oklch(0.5544 0.0407 257.42)` · applied `oklch(0.5461 0.2152 262.88)` · phone `oklch(0.5413 0.2466 293.01)` · interview `oklch(0.6658 0.1574 58.32)` · offer `oklch(0.596 0.1274 163.23)` · rejected `oklch(0.5771 0.2152 27.33)`
- Hex equivalents: saved `#64748b` · applied `#2563eb` · phone `#7c3aed` · interview `#d97706` · offer `#059669` · rejected `#dc2626`

> Status chips derive their background and text from the status's own hex at runtime:
> `background: color-mix(in srgb, <hex> 14%, white)`, `color: color-mix(in srgb, <hex> 78%, black)`. Custom user statuses render consistently this way.

## 3. Typography

- **UI / body / display**: **Plus Jakarta Sans** (`--font-sans`), weights 400/500/600/700. There is no separate display face — hierarchy comes from size and weight, not a second family.
- **Data / numerics / IDs / source tags / dates**: **JetBrains Mono** (`--font-mono`), weights 400/500, with `tabular-nums`.
- Base body is **14px / 1.5** on the slate-900 foreground.

Scale (resolved from the Tailwind utilities actually used — sizes are Tailwind defaults, not custom tokens):
| Utility | Size | Use |
|---|---|---|
| `text-[10px]` | 10px | sidebar section labels (uppercase, tracked) |
| `text-xs` | 12px | badges, table heads, scraped dates, field labels |
| `text-sm` | 14px | body, buttons, nav, breadcrumb, table cells |
| `text-base` | 16px | card titles (semibold, snug) |
| `text-lg` | 18px | page titles — Jobs / Applications / Statuses (bold) |
| `text-xl` | 20px | auth screen headings (login / signup) |

Weights: regular 400, medium 500, semibold 600, bold 700. Tracking: `tracking-tight`
(−0.025em) on the brand wordmark and page titles; `tracking-wide` (0.025em) on
uppercase table heads and the source badge; `tracking-wider` (0.05em) on sidebar
section labels.

## 4. Spacing

4px base step. Common rhythm: nav links `px-2.5 py-1.5`, table cells `px-4 py-2.5`,
card padding `p-5` (header `pb-3`, footer `pb-5`), page gutter `px-7 py-6`, topbar
gutter `px-6`, gaps of `0.5rem`–`0.625rem`. Density is tight but breathable —
admin-console, not airy marketing.

Fixed dimensions: controls `h-9` (36px) default, `h-8` (32px) small/icon; topbar and
sidebar header `h-14` (56px); sidebar `13.75rem` expanded / `3.5rem` collapsed.

Radius (Tailwind utilities): `rounded-md` 6px (buttons, fields, nav links, icon
buttons), `rounded-lg` 8px (dropdown menus), `rounded-xl` 12px (cards, table
containers, list panels), `rounded-2xl` 16px (modals, auth cards), `rounded-full`
(pills, avatars, status dots).

Elevation: cards, tables, and the shell rely on **borders only**. Shadows are
reserved for overlays — `shadow-xl` on dropdown content and the auth cards,
`shadow-2xl` on modals (over a `bg-black/30` + `backdrop-blur-[2px]` scrim).

## 5. Layout & Composition

App shell = **fixed dark sidebar + main column**. Sidebar (`#111921`) is collapsible
(220px ⇆ 56px) with a brand lockup at top, section groups (`Main`, `Settings`), and
nav links that use a teal-tinted active pill. Main column is a 56px topbar (breadcrumb
left `Job Scraper / <Page>`, circular soft-teal avatar right) over a scrolling content
region on the light canvas. Primary surfaces: the **Jobs data table** (search +
sortable columns + pagination, 20/page), the **Applications tracker**, and
**Settings → Statuses**. Tables are full-width with hairline row borders, a muted
header band, uppercase faint heads, and hover-row tinting. Slim 6px scrollbars.

## 6. Components

- **Button** (`rounded-md`, `h-9`, `text-sm` medium): `default` solid teal → teal-hover; `destructive` red; `outline` (border + surface, hover strengthens border/text); `secondary` "soft teal" (accent-subtle bg + accent-border + accent-text); `ghost`; `link`. Sizes `sm` / `default` / `lg` plus square icon variants; `focus-visible` teal ring with offset.
- **Card** (`rounded-xl`, border, no shadow): header `p-5 pb-3`, base-size semibold title, content `px-5`, footer `px-5 pb-5` pushed to bottom.
- **Badge** (`rounded-full`, `text-xs` medium): `default` solid teal, `secondary` muted, `outline`, and `source` — mono uppercase tracked slate chip for the job-board origin.
- **StatusBadge**: pill with a coloured dot + `color-mix` soft tint derived from the status hex (see §2).
- **Table**: wrapper scrolls x; `text-sm`; muted header band; `th` = `h-9` uppercase `text-xs` faint tracked; `td` = `px-4 py-2.5`; rows hairline-bordered, hover `surface-muted`, last row borderless.
- **Dropdown menu**: `rounded-lg` surface card, `min-w-[10rem]`, `shadow-xl`, animated in/out; items with leading 14px icons; used for per-row job actions (Apply ↗, Track / Edit application).
- **Field** (`.field` / `.field-label`): bordered surface input, teal focus border + soft teal ring (`focus:ring-primary/10`); small muted medium label above.
- **Icons**: inline stroke SVGs (Lucide-style, `stroke-width:2`), 14–16px, `currentColor` — never emoji.

## 7. Motion & Interaction

`transition-colors` / `transition-all` on interactive elements; sidebar collapse is a
300ms `width` transition. Dropdown content animates with fade + 95% zoom on
open/close (`tailwindcss-animate`). Focus is always a visible teal ring
(`ring-2 ring-primary`, offset 1). Hover states: rows tint to `surface-muted`,
outline buttons strengthen border + text, ghost/icon buttons pick up
`accent-subtle`. Keep transitions short and functional; honour `prefers-reduced-motion`
by disabling non-essential animation.

## 8. Voice & Brand

Product name **Job Scraper**. Copy is plain, operator-facing, sentence case:
"Search by role or company…", "Track application", "Edit application". Row actions
read as short verbs ("Apply ↗", "Track application", "Edit", "Delete"). Column heads
are short nouns (Title, Company, Location, Source, Scraped, Status). Dates render
`en-GB` short ("8 Jun 2026") in mono. Empty values are a faint em dash `—`, never
"N/A". No marketing superlatives or invented metrics.

## 9. Anti-patterns

- ❌ Dark mode, or any canvas other than `#f8fafc` / white surfaces.
- ❌ Colours outside this palette; more than ~two teal moments per screen.
- ❌ Gradients, drop shadows on cards, hero images, decorative illustration.
- ❌ Emoji icons or an icon beside every heading — use inline stroke SVGs only.
- ❌ A second display typeface, or body text below 14px / table text below 12px.
- ❌ Generic "N/A" or invented stats; use a faint `—` for missing data.
- ❌ Replacing mono for data/IDs/dates with the sans face (lose `tabular-nums`).
- ❌ Sidebar in any colour but the near-black `#111921` panel scale.
