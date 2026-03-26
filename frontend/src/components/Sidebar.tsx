import { createSignal } from "solid-js";
import { Link, useLocation, useNavigate } from "@tanstack/solid-router";
import { logout } from "../api/auth";
import { Show } from "solid-js";

export default function Sidebar() {
  const [expanded, setExpanded] = createSignal(true);
  const location = useLocation();
  const isJobsActive = () => location.pathname === "/jobs";
  const isApplicationsActive = () => location.pathname === "/applications";
  const isStatusesActive = () => location.pathname === "/settings/statuses";
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate({ to: "/login" });
  };

  return (
    <aside
      class="flex flex-col h-screen border-r border-[var(--line)] bg-[var(--surface-strong)] backdrop-blur-sm shrink-0 transition-[width] duration-300 overflow-hidden"
      style={{ width: expanded() ? "14rem" : "3.5rem" }}
    >
      {/* Brand */}
      <div class="flex items-center gap-3 px-3 h-14 border-b border-[var(--line)] shrink-0">
        <div class="shrink-0 w-8 h-8 rounded-lg bg-[rgba(79,184,178,0.18)] flex items-center justify-center text-[var(--lagoon-deep)]">
          <svg
            aria-hidden="true"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <rect x="2" y="7" width="20" height="14" rx="2" />
            <path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2" />
          </svg>
        </div>
        {expanded() && (
          <span class="font-semibold text-sm text-[var(--sea-ink)] whitespace-nowrap">
            Job Scraper
          </span>
        )}
      </div>

      {/* Nav */}
      <nav class="flex-1 px-2 py-3">
        <Link
          to="/jobs"
          search={{ page: 1 }}
          title="Jobs"
          class={`flex items-center gap-3 px-2 py-2 rounded-lg text-sm transition-colors ${
            isJobsActive()
              ? "bg-[rgba(79,184,178,0.18)] text-[var(--sea-ink)] font-medium"
              : "text-[var(--sea-ink-soft)] hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]"
          }`}
        >
          <svg
            aria-hidden="true"
            class="shrink-0"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <rect x="2" y="7" width="20" height="14" rx="2" />
            <path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2" />
          </svg>
          {expanded() && <span class="whitespace-nowrap">Jobs</span>}
        </Link>

        <Link
          to="/applications"
          title="Applications"
          class={`flex items-center gap-3 px-2 py-2 rounded-lg text-sm transition-colors ${
            isApplicationsActive()
              ? "bg-[rgba(79,184,178,0.18)] text-[var(--sea-ink)] font-medium"
              : "text-[var(--sea-ink-soft)] hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]"
          }`}
        >
          <svg
            aria-hidden="true"
            class="shrink-0"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <polyline points="14 2 14 8 20 8" />
            <line x1="16" y1="13" x2="8" y2="13" />
            <line x1="16" y1="17" x2="8" y2="17" />
            <polyline points="10 9 9 9 8 9" />
          </svg>
          {expanded() && <span class="whitespace-nowrap">Applications</span>}
        </Link>

        <Link
          to="/settings/statuses"
          title="Status settings"
          class={`flex items-center gap-3 px-2 py-2 rounded-lg text-sm transition-colors ${
            isStatusesActive()
              ? "bg-[rgba(79,184,178,0.18)] text-[var(--sea-ink)] font-medium"
              : "text-[var(--sea-ink-soft)] hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]"
          }`}
        >
          <svg
            aria-hidden="true"
            class="shrink-0"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
          </svg>
          {expanded() && <span class="whitespace-nowrap">Statuses</span>}
        </Link>
      </nav>

      {/* Bottom actions */}
      <div class="px-2 py-3 border-t border-[var(--line)] shrink-0 flex flex-col gap-1">
        <button
          type="button"
          onClick={handleLogout}
          title="Sign out"
          class="flex items-center gap-3 px-2 py-2 w-full rounded-lg text-sm text-[var(--sea-ink-soft)] transition-colors hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]"
        >
          <svg
            aria-hidden="true"
            class="shrink-0"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
          {expanded() && (
            <span class="whitespace-nowrap text-xs">Sign out</span>
          )}
        </button>
        <button
          type="button"
          onClick={() => setExpanded((e) => !e)}
          title={expanded() ? "Collapse sidebar" : "Expand sidebar"}
          class="flex items-center gap-3 px-2 py-2 w-full rounded-lg text-sm text-[var(--sea-ink-soft)] transition-colors hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]"
        >
          <svg
            aria-hidden="true"
            class="shrink-0 transition-transform duration-300"
            style={{
              transform: expanded() ? "rotate(0deg)" : "rotate(180deg)",
            }}
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <polyline points="15 18 9 12 15 6" />
          </svg>
          {expanded() && (
            <span class="whitespace-nowrap text-xs">Collapse</span>
          )}
        </button>
      </div>
    </aside>
  );
}
