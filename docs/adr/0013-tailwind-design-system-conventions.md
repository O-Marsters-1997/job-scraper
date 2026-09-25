# ADR 0013 — Tailwind / shadcn-solid design-system conventions

- **Variants live in `cva`** when a component has two or more mutually exclusive styles that a caller would choose between. Single-style components take overrides through `cn()`; forcing `cva` on them is ceremony.
- **Text sizes come from named `--text-*` tokens** in `@theme` (`frontend/src/styles.css`). Arbitrary `text-[Npx]` values aren't allowed: add a token. Genuinely one-off layout values (grid columns, a single max-width) may stay arbitrary.
- Colour stays on semantic `--color-*` OKLCH tokens, with no hex or raw Tailwind colour utilities in `className`.
