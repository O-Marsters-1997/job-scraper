# frontend

SolidJS SPA for the job-scraper pipeline. Displays scraped jobs, AI suitability scores, application tracking, CV templates, and insights charts.

## Stack

- **SolidJS** — fine-grained reactive UI (components run once, no VDOM)
- **TanStack Router** — file-based routing with loaders
- **TanStack Query** — server state, caching, background refetch
- **TanStack Table** — jobs data table
- **Vite** — build and dev server
- **Tailwind CSS v4** — utility-first styling via CSS `@theme`
- **Biome** — lint + format (tabs, double quotes)
- **Kobalte** — accessible headless primitives (Dialog, Switch, Popover…)

## Development

```bash
bun install

# dev against real API (backend must be running on :8080)
bun run dev          # → http://localhost:3000

# dev with mock backend — no Go process needed
bun run dev:test     # → http://localhost:4444
```

Set `VITE_MOCK=true` to force mock mode in any env (already set for `dev:test`).

## Scripts

| Command | What it does |
|---------|-------------|
| `bun run dev` | Dev server on :3000, proxies API to :8080 |
| `bun run dev:test` | Dev server on :4444 with in-browser mock backend |
| `bun run build` | Vite production build → `dist/` |
| `bun run typecheck` | TypeScript check (tsgo, no emit) |
| `bun run check` | Biome lint + format check |
| `bun run test` | Run all `*.check.ts` self-tests |
| `bun run e2e` | Playwright end-to-end tests |

## Directory layout

```
src/
  api/          API fetch functions + Zod schemas (trust boundary)
  components/   Shared UI components
    cv/         CV-specific components (AddDocDialog)
    insights/   Per-chart components for the Insights page
    jobs/       Job-detail components (SuitabilityPanel, TrackApplicationDialog…)
    ui/         Design-system primitives (Button, Card, Dialog, Input…)
  hooks/        TanStack Query hooks (useJobs, useApplications…)
  lib/          Pure utilities (charts, color, datetime, tweaks data/apply)
  mocks/        In-browser mock backend (MSW + mock DB)
  routes/       File-based routes
    _auth/      Authenticated routes (jobs, applications, insights, settings…)
  types/        Hand-mirrored Go struct types
```

## Architecture notes

- **Mock backend**: `src/mocks/` uses MSW to intercept fetch calls. Enabled via `VITE_MOCK`. Mirrors the real API surface so the SPA is fully functional offline.
- **Trust boundary**: `api/client.ts` (`apiFetch`) accepts an optional Zod schema and `.parse()`s the response. Schemas adopted for high-value endpoints (`/scoring-config`).
- **Async pattern**: `QueryBoundary` wraps loading/error/empty/success states; all authenticated routes go through it.
- **Edit-safe forms**: Form state lives in child components that mount once when data is available (`<QueryBoundary>{(data) => <Form data={data} />}</QueryBoundary>`). Background refetches update the cache but don't remount the form.
- **Tweaks**: `lib/tweaks.ts` exports data/types; `lib/tweaks.apply.ts` exports DOM-mutating `apply*` functions. Applied at app startup from `localStorage`.

## Testing

`*.check.ts` files are zero-dependency self-tests runnable via `bun`:

```bash
bun run test              # all checks via scripts/run-checks.ts
bun src/api/client.check.ts   # single check file
```

E2E tests cover login, logout, and job browsing:

```bash
bun run e2e
```
