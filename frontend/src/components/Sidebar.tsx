import { createSignal, Show } from "solid-js";
import { Link, useLocation, useNavigate } from "@tanstack/solid-router";
import { logout } from "../api/auth";
import { cn } from "../lib/utils";

export default function Sidebar() {
	const [expanded, setExpanded] = createSignal(true);
	const location = useLocation();
	const isJobsActive = () => location().pathname === "/jobs";
	const isApplicationsActive = () => location()?.pathname === "/applications";
	const isStatusesActive = () => location().pathname === "/settings/statuses";
	const navigate = useNavigate();

	const handleLogout = async () => {
		await logout();
		navigate({ to: "/login" });
	};

	const navLink = (active: boolean) =>
		cn(
			"flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors",
			active
				? "bg-sidebar-active text-sidebar-active-foreground"
				: "text-sidebar-foreground hover:bg-sidebar-hover hover:text-sidebar-foreground-strong",
		);

	return (
		<aside
			class="flex h-screen shrink-0 flex-col overflow-hidden border-r border-sidebar-border bg-sidebar transition-[width] duration-300"
			style={{ width: expanded() ? "13.75rem" : "3.5rem" }}
		>
			{/* Brand */}
			<div class="flex h-14 shrink-0 items-center gap-2.5 border-b border-sidebar-border px-4">
				<div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-sidebar-active text-sidebar-active-foreground">
					<svg
						aria-hidden="true"
						width="14"
						height="14"
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
				<Show when={expanded()}>
					<span class="text-sm font-semibold tracking-tight text-sidebar-foreground-strong whitespace-nowrap">
						Job Scraper
					</span>
				</Show>
			</div>

			{/* Nav */}
			<nav class="flex flex-1 flex-col gap-0.5 overflow-y-auto px-2 py-3">
				<Show when={expanded()}>
					<span class="px-2.5 pt-2.5 pb-1 text-[10px] font-semibold tracking-wider text-sidebar-foreground/60 uppercase">
						Main
					</span>
				</Show>

				<Link to="/jobs" title="Jobs" class={navLink(isJobsActive())}>
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
					<Show when={expanded()}>
						<span class="whitespace-nowrap">Jobs</span>
					</Show>
				</Link>

				<Link
					to="/applications"
					title="Applications"
					class={navLink(isApplicationsActive())}
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
					<Show when={expanded()}>
						<span class="whitespace-nowrap">Applications</span>
					</Show>
				</Link>

				<Show when={expanded()}>
					<span class="px-2.5 pt-4 pb-1 text-[10px] font-semibold tracking-wider text-sidebar-foreground/60 uppercase">
						Settings
					</span>
				</Show>

				<Link
					to="/settings/statuses"
					title="Status settings"
					class={navLink(isStatusesActive())}
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
					<Show when={expanded()}>
						<span class="whitespace-nowrap">Statuses</span>
					</Show>
				</Link>
			</nav>

			{/* Bottom actions */}
			<div class="flex shrink-0 flex-col gap-0.5 border-t border-sidebar-border px-2 py-2">
				<button
					type="button"
					onClick={handleLogout}
					title="Sign out"
					class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-hover hover:text-sidebar-foreground-strong"
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
					<Show when={expanded()}>
						<span class="whitespace-nowrap">Sign out</span>
					</Show>
				</button>
				<button
					type="button"
					onClick={() => setExpanded((e) => !e)}
					title={expanded() ? "Collapse sidebar" : "Expand sidebar"}
					aria-label={expanded() ? "Collapse sidebar" : "Expand sidebar"}
					class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-hover hover:text-sidebar-foreground-strong"
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
					<Show when={expanded()}>
						<span class="whitespace-nowrap">Collapse</span>
					</Show>
				</button>
			</div>
		</aside>
	);
}
