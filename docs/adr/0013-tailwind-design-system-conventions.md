# ADR 0013 — Tailwind / shadcn-solid design-system conventions

**Status:** Accepted  
**Date:** 2026-06-24

## Context

The frontend uses SolidJS with shadcn-solid, Kobalte, and Tailwind v4 (CSS-first `@theme`). After consolidating the violet design system (ADR prior work), a follow-up audit found two discipline gaps:

1. **Variant management was split-brain.** `button`, `badge`, and `label` used `cva`; several components with genuine variant axes (sidebar nav link, sidebar count badge, TweaksPanel selectable buttons, source badge) hand-rolled their variants as inline ternaries and `Record` lookups.

2. **The type scale had drifted.** Six near-duplicate sub-12px text sizes (`text-[9.5px]` through `text-[11.5px]`) and several one-off display sizes were scattered as arbitrary Tailwind values across the codebase, despite `DESIGN.md` documenting a clean named scale.

## Decision

### 1. Variants live in `cva`; static wrappers stay `cn()`

`cva` (class-variance-authority) is used for any component with **two or more mutually exclusive style states** — i.e., a real variant axis. The test: "would a consumer reasonably want to select between these styles at call-site?" If yes, it's a `cva` variant. If a component has one style and just accepts an override via `cn()`, it stays as `cn()`.

Applied to:
- `Sidebar`: `navLinkVariants` (active/idle) + `sidebarBadgeVariants` (active/idle)
- `TweaksPanel`: `themeButtonVariants`, `themeButtonLabelVariants`, `selectableButtonVariants` (selected/unselected for font and size pickers)
- `SourceBadge`: `sourceColorVariants` (one variant per known job board)

Not applied to: `Table`, `Dialog`, `Select`, `Sheet`, `Card`, `Input`, `Switch`, `Skeleton` — these have one style and no meaningful variant axis. Forcing `cva` on them is empty ceremony.

### 2. All sizing comes from named `@theme` tokens; arbitrary values are a smell

Every text size is defined as a named `--text-*` token in `frontend/src/styles.css` under `@theme`. Arbitrary `text-[Npx]` values are not permitted; add a named token to `@theme` instead.

The type scale was extended with:
- `text-2xs` = 10px — sidebar/panel micro-labels, badge labels (consolidates former 9.5/10/10.5px)
- `text-data` = 13px — table cells, panel titles (between xs=12 and sm=14)
- `text-auth-heading`, `text-auth-subtext`, `text-auth-action` — login/signup form surface
- `text-display`, `text-display-prose` — auth brand panel (line-height bundled in token)
- Former 11/11.5px text consolidated to `text-xs` (12px)

One-off **layout** dimensions that are genuinely single-use (grid-cols, single-use max-widths, scrim blur) may remain as arbitrary values — the constraint applies to the type scale, not to structural layout.

### 3. Color and spacing: no change

The existing token discipline (`--color-*` OKLCH semantic tokens, no hardcoded hex in `className`, no raw Tailwind color utilities) was already correct and is preserved.

## Consequences

- All variant logic is now co-located in named `cva` blocks, making every variant axis discoverable and type-safe.
- The type scale is fully named; adding new text sizes requires a token in `@theme`, preventing further drift.
- `DESIGN.md` typography table now references named utilities and includes a rule against arbitrary text values.
- The six near-duplicate sub-12px sizes were consolidated to two steps (`text-2xs` at 10px, `text-xs` at 12px) — a deliberate, screenshot-verified visual normalisation.
