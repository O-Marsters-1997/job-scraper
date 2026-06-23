import { Link, useLocation, useNavigate } from "@tanstack/solid-router";
import { createEffect, createSignal, onCleanup, onMount, Show } from "solid-js";
import { FastTrackMark } from "@/components/brand-mark";
import { cn } from "@/lib/utils";
import { logout } from "../api/auth";
import { useApplications } from "../hooks/useApplications";
import { useGoogleStatus } from "../hooks/useGoogle";
import { useJobs } from "../hooks/useJobs";

interface SidebarProps {
	// Below the md breakpoint the sidebar is an off-canvas drawer driven by the
	// app shell; on md+ it's an in-flow, collapsible rail.
	mobileOpen?: boolean;
	onMobileClose?: () => void;
}

export default function Sidebar(props: SidebarProps) {
	const [expanded, setExpanded] = createSignal(true);
	const location = useLocation();
	// Drawer always shows labels regardless of the desktop collapse state.
	const showLabels = () => props.mobileOpen || expanded();

	// Close the drawer on navigation so a tapped link doesn't leave it covering
	// the page it just opened.
	createEffect(() => {
		location().pathname;
		props.onMobileClose?.();
	});

	onMount(() => {
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape") props.onMobileClose?.();
		};
		window.addEventListener("keydown", onKey);
		onCleanup(() => window.removeEventListener("keydown", onKey));
	});
	const isOverviewActive = () => location().pathname === "/overview";
	const isJobsActive = () => location().pathname === "/jobs";
	const isApplicationsActive = () => location()?.pathname === "/applications";
	const isInsightsActive = () => location().pathname === "/insights";
	const isStatusesActive = () => location().pathname === "/settings/statuses";
	const isSearchesActive = () => location().pathname === "/settings/searches";
	const isIntegrationsActive = () =>
		location().pathname === "/settings/integrations";
	const isCVTemplatesActive = () => location().pathname === "/cv-templates";
	const navigate = useNavigate();

	const jobsQuery = useJobs();
	const appsQuery = useApplications();
	const googleStatus = useGoogleStatus();

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

	const badgeClass = (active: boolean) =>
		cn(
			"ml-auto rounded-full px-1.5 py-0.5 font-mono text-[10.5px] tabular-nums",
			active
				? "bg-sidebar-badge-active text-sidebar-active-foreground"
				: "bg-sidebar-badge text-sidebar-foreground",
		);

	return (
		<>
			<Show when={props.mobileOpen}>
				<button
					type="button"
					aria-label="Close menu"
					onClick={() => props.onMobileClose?.()}
					class="fixed inset-0 z-40 bg-black/40 md:hidden"
				/>
			</Show>
			<aside
				class={cn(
					"fixed inset-y-0 left-0 z-50 flex h-screen w-[13.75rem] flex-col overflow-hidden border-r border-sidebar-border bg-sidebar",
					"transition-transform duration-300 md:relative md:z-auto md:translate-x-0 md:shrink-0 md:transition-[width]",
					props.mobileOpen ? "translate-x-0" : "-translate-x-full",
					expanded() ? "md:w-[13.75rem]" : "md:w-14",
				)}
			>
				<div class="flex h-14 shrink-0 items-center gap-2.5 border-b border-sidebar-border px-4">
					<div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-sidebar-active text-sidebar-active-foreground">
						<FastTrackMark size={15} />
					</div>
					<Show when={showLabels()}>
						<span class="whitespace-nowrap text-sm font-semibold tracking-tight text-sidebar-foreground-strong">
							FastTrack
						</span>
					</Show>
				</div>

				<nav class="flex flex-1 flex-col gap-0.5 overflow-y-auto px-2 py-3">
					<Show when={showLabels()}>
						<span class="px-2.5 pb-1 pt-2.5 text-[10px] font-semibold uppercase tracking-wider text-sidebar-foreground/60">
							Main
						</span>
					</Show>

					<Link
						to="/overview"
						title="Overview"
						class={navLink(isOverviewActive())}
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
							<rect width="7" height="9" x="3" y="3" rx="1" />
							<rect width="7" height="5" x="14" y="3" rx="1" />
							<rect width="7" height="9" x="14" y="12" rx="1" />
							<rect width="7" height="5" x="3" y="16" rx="1" />
						</svg>
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Overview</span>
						</Show>
					</Link>

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
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Jobs</span>
							<Show when={(jobsQuery.data?.length ?? 0) > 0}>
								<span class={badgeClass(isJobsActive())}>
									{jobsQuery.data?.length}
								</span>
							</Show>
						</Show>
					</Link>

					<Link
						to="/applications"
						search={{ status: undefined }}
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
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Applications</span>
							<Show when={(appsQuery.data?.length ?? 0) > 0}>
								<span class={badgeClass(isApplicationsActive())}>
									{appsQuery.data?.length}
								</span>
							</Show>
						</Show>
					</Link>

					{/* CVs — only shown when Google is connected */}
					<Show when={googleStatus.data?.connected}>
						<Link
							to="/cv-templates"
							title="CVs"
							class={navLink(isCVTemplatesActive())}
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
								<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
								<polyline points="14 2 14 8 20 8" />
								<line x1="16" y1="13" x2="8" y2="13" />
								<line x1="16" y1="17" x2="8" y2="17" />
							</svg>
							<Show when={showLabels()}>
								<span class="whitespace-nowrap">CVs</span>
							</Show>
						</Link>
					</Show>
					{/* Insights */}
					<Link
						to="/insights"
						title="Insights"
						class={navLink(isInsightsActive())}
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
							<line x1="18" y1="20" x2="18" y2="10" />
							<line x1="12" y1="20" x2="12" y2="4" />
							<line x1="6" y1="20" x2="6" y2="14" />
						</svg>
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Insights</span>
						</Show>
					</Link>

					<Show when={showLabels()}>
						<span class="px-2.5 pb-1 pt-4 text-[10px] font-semibold uppercase tracking-wider text-sidebar-foreground/60">
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
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Statuses</span>
						</Show>
					</Link>

					<Link
						to="/settings/searches"
						title="Tracked searches"
						class={navLink(isSearchesActive())}
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
							<circle cx="11" cy="11" r="8" />
							<line x1="21" y1="21" x2="16.65" y2="16.65" />
						</svg>
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Searches</span>
						</Show>
					</Link>

					<Link
						to="/settings/integrations"
						title="Integrations"
						class={navLink(isIntegrationsActive())}
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
							<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
							<path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
						</svg>
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Integrations</span>
						</Show>
					</Link>
				</nav>

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
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Sign out</span>
						</Show>
					</button>
					<button
						type="button"
						onClick={() => setExpanded((e) => !e)}
						title={expanded() ? "Collapse sidebar" : "Expand sidebar"}
						aria-label={expanded() ? "Collapse sidebar" : "Expand sidebar"}
						class="hidden w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-hover hover:text-sidebar-foreground-strong md:flex"
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
						<Show when={showLabels()}>
							<span class="whitespace-nowrap">Collapse</span>
						</Show>
					</button>
				</div>
			</aside>
		</>
	);
}
