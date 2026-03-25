import { createSignal } from 'solid-js'
import { Link, useMatch } from '@tanstack/solid-router'

export default function Sidebar() {
  const [expanded, setExpanded] = createSignal(true)
  const isJobsActive = useMatch(() => ({ from: '/jobs', strict: false }))

  return (
    <aside
      class="flex flex-col h-screen border-r border-[var(--line)] bg-[var(--surface-strong)] backdrop-blur-sm shrink-0 transition-[width] duration-300 overflow-hidden"
      style={{ width: expanded() ? '14rem' : '3.5rem' }}
    >
      {/* Brand */}
      <div class="flex items-center gap-3 px-3 h-14 border-b border-[var(--line)] shrink-0">
        <div class="shrink-0 w-8 h-8 rounded-lg bg-[rgba(79,184,178,0.18)] flex items-center justify-center text-[var(--lagoon-deep)]">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="7" width="20" height="14" rx="2"/>
            <path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/>
          </svg>
        </div>
        {expanded() && (
          <span class="font-semibold text-sm text-[var(--sea-ink)] whitespace-nowrap">Job Scraper</span>
        )}
      </div>

      {/* Nav */}
      <nav class="flex-1 px-2 py-3">
        <Link
          to="/jobs"
          title="Jobs"
          class={`flex items-center gap-3 px-2 py-2 rounded-lg text-sm transition-colors ${
            isJobsActive()
              ? 'bg-[rgba(79,184,178,0.18)] text-[var(--sea-ink)] font-medium'
              : 'text-[var(--sea-ink-soft)] hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]'
          }`}
        >
          <svg class="shrink-0" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="7" width="20" height="14" rx="2"/>
            <path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/>
          </svg>
          {expanded() && <span class="whitespace-nowrap">Jobs</span>}
        </Link>
      </nav>

      {/* Toggle */}
      <div class="px-2 py-3 border-t border-[var(--line)] shrink-0">
        <button
          type="button"
          onClick={() => setExpanded(e => !e)}
          title={expanded() ? 'Collapse sidebar' : 'Expand sidebar'}
          class="flex items-center gap-3 px-2 py-2 w-full rounded-lg text-sm text-[var(--sea-ink-soft)] transition-colors hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]"
        >
          <svg
            class="shrink-0 transition-transform duration-300"
            style={{ transform: expanded() ? 'rotate(0deg)' : 'rotate(180deg)' }}
            width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
          >
            <polyline points="15 18 9 12 15 6"/>
          </svg>
          {expanded() && <span class="whitespace-nowrap text-xs">Collapse</span>}
        </button>
      </div>
    </aside>
  )
}
