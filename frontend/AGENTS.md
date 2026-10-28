# frontend/AGENTS.md

Solid (not React) — `@tanstack/solid-query`, `@tanstack/solid-router`, Kobalte primitives.
Read `../DESIGN.md` before any UI change; it's the source of truth for colours, type, spacing.

## Patterns

- **API call + hook** — copy `src/api/applicationStatuses.ts` (fetch functions) and
  `src/hooks/useProfile.ts` (`queryOptions`, `createQuery`, `useInvalidatingMutation`
  from `src/hooks/useInvalidatingMutation.ts`).
  - Every new api function wraps its call in `mocked(...)` (`src/api/config.ts`), backed by `src/mocks/db.ts`.
  - Parse responses with a zod schema (see `scores.ts`, `scoringConfig.ts`); don't just cast.
  - Shared types live in `src/types/*.ts`, not declared inline in the api file.
- **Unit check** — a `src/**/*.check.ts` file that throws at import time (copy
  `src/lib/jobFilters.check.ts`). There's no test framework; `bun run test` runs every `*.check.ts`.
- **Styling**
  - Use `cva` when a component has two or more mutually exclusive variants a caller picks
    between; single-style components take overrides through `cn()`.
  - Text sizes come from named `--text-*` tokens in `@theme` (`src/styles.css`). No arbitrary
    `text-[Npx]`: add a token. One-off layout values (grid columns, a single max-width) may stay
    arbitrary.
  - Colour uses semantic `--color-*` tokens only, with no hex or raw Tailwind colour utilities.
- **Loading and errors** — TanStack Router runs each navigation in a Solid transition, and
  solid-query's `data` getter is a Suspense resource read while there is no data, so a stray
  read holds the old page or blanks the layout.
  1. Page or card data from a query goes through `QueryBoundary` (`notFound` for empty,
     `fallbackRows` matching the loaded shape). Use plain `Switch`/`Match` only for several
     distinct error states (e.g. `ExperienceMatch`).
  2. Never read `query.data` in render before checking `isPending`/`isSuccess`; use
     `loadedData(query)` (`src/lib/queryData.ts`). Don't add `<Suspense>` for queries; the
     router owns the boundary (`defaultPendingComponent`, plus one around `<Outlet />`).
  3. Mutations: `loading={m.isPending}` on the triggering `Button` (spinner, disabled,
     fixed width); keep the label static. Icon-only controls use `disabled`.
  4. Loading data gets a skeleton, never "Loading…" text. The `Spinner` is only for
     shell-less loads (`FullPageSpinner` on the `_auth` pending state) and `Button`. Text is only for
     validate-as-you-type (`PasteBox`).
  5. Refetches keep stale data (`keepPreviousData`), with no indicator.
  6. A route with data gets a `loader` that `prefetchQuery`s it, so `defaultPreload: "intent"`
     warms the cache on hover.- **E2E** — page object in `e2e/src/pages/`, spec in `e2e/tests/` (copy an existing pair, e.g.
  `companies.page.ts` + `companies.spec.ts`).

## Commands

See `package.json` scripts (`bun run <script>`) for lint, typecheck, test, and build.
E2E: `bunx playwright test` (starts vite on :4444 with `VITE_MOCK=true`).

## Gotchas

- E2E runs against `VITE_MOCK=true` with no backend. Don't add an api function used by an
  e2e-covered page without a `useMocks()` branch; do add fake data to `src/mocks/db.ts`.
- UI primitives (`src/components/ui/`) are vendored from the Zaidan registry. Don't use the
  shadcn CLI or hand-patch them; do run `bun run add-component <name>` to add or update one.
  Exceptions: `ui/spinner.tsx` is local, and `ui/button.tsx` carries a local `loading` prop;
  re-apply it after any `add-component button`.
