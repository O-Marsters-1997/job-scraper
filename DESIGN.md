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
and trustworthy** — a quiet light canvas (`#f8fafc`) holding white cards, framed on
the left by a single near-black sidebar (`#111921`). One teal brand accent
(`#0f9d92`) carries every primary action and active state; everything else is
slate-neutral. The feeling is "admin console you live in all day": shadcn / Tabler /
TailAdmin light-mode lineage — information per square inch over vibes, with just
enough colour to signal state. No gradients, no hero imagery, no decoration for its
own sake.

## 2. Color

Light mode only. Borders and surface tints — not shadows — do the structural work.

**Canvas & surfaces**
- `--color-background` `#f8fafc` — app canvas
- `--color-surface` `#ffffff` — cards, panels, inputs, menus
- `--color-surface-muted` `#fafbfc` — table headers, hover rows, secondary chips

**Text**
- `--color-foreground` `#0f172a` — primary text, headings, cell values
- `--color-muted` `#475569` — secondary text, company/location cells
- `--color-faint` `#94a3b8` — metadata, placeholders, empty `—`, table heads

**Lines**
- `--color-border` `#e2e8f0` — default hairline
- `--color-border-strong` `#cbd5e1` — hover borders, breadcrumb divider

**Teal brand accent** (used at most twice per screen)
- `--color-primary` `#0f9d92` / hover `--color-primary-hover` `#0a7b72`
- `--color-primary-foreground` `#ffffff`
- `--color-accent-subtle` `#f0fdfa` · `--color-accent-border` `#99f6e4` · `--color-accent-text` `#134e4a` — secondary/"soft teal" buttons, avatar chip, ghost hovers

**Dark sidebar panel** (its own self-contained scale)
- `--color-sidebar` `#111921` · hover `--color-sidebar-hover` `#1b2a38` · border `--color-sidebar-border` `#1b2a38`
- foreground `--color-sidebar-foreground` `#7a8fa6` → strong `--color-sidebar-foreground-strong` `#dce8f0`
- active background `--color-sidebar-active` `rgba(20,184,166,0.13)` · active foreground `--color-sidebar-active-foreground` `#5eead4`

**Danger**
- `--color-destructive` `#ef4444` / strong `--color-destructive-strong` `#dc2626` / subtle `--color-destructive-subtle` `#fef2f2`

**Application-status palette** (chip defaults; tinted via `color-mix`)
- saved `#64748b` · applied `#2563eb` · phone `#7c3aed` · interview `#d97706` · offer `#059669` · rejected `#dc2626`

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
