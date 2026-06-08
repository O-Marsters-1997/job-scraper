# Product

## Register

product

## Users

Job hunters (initially just the owner and a small circle of peers) running active startup job searches. They use this during focused job-search sessions: scanning fresh listings, marking ones to apply to, logging application state, and checking pipeline health. The primary interaction is the jobs data table: read-heavy, occasional writes.

## Product Purpose

An automated personal job-hunting command centre. The backend crawls startup job boards on a schedule; the frontend surfaces the results and lets users manage their application pipeline from a single view. Success looks like: every live listing visible at a glance, application state always current, and zero friction moving a job through the pipeline.

## Brand Personality

Precise, composed, purposeful. This is software that respects your attention. The interface should feel like a well-made internal tool that happens to look good: clean tables, tight typography, neutral palette, no decorative noise. Think shadcn/ui discipline: components that know their place.

## Anti-references

- LinkedIn and Indeed: corporate job board density, grey monotony, zero personality.
- Generic AI tool aesthetic: cream or sand backgrounds, gradient text, hero metric tiles, heavy glassmorphism used as decoration.
- SaaS marketing energy: anything that looks like it is trying to impress rather than serve.

## Design Principles

1. **The table is the interface.** Data density is a feature, not a problem. Layouts exist to hold data, not the other way around.
2. **No chrome for chrome's sake.** Every UI element must earn its presence. If it doesn't help the user read, navigate, or act, remove it.
3. **Legibility at density.** High information per pixel, without sacrificing contrast, spacing rhythm, or scanability.
4. **Quiet confidence.** The palette and typography signal precision, not personality. Color is used for state and action, not aesthetics.
5. **Accessible by default.** WCAG AA minimum (4.5:1 body, 3:1 large text). Keyboard-navigable tables and modals. Reduced-motion respect throughout.

## Accessibility & Inclusion

WCAG AA baseline. All interactive elements keyboard accessible. Focus rings visible at all times. Reduced-motion alternative for every animation.
