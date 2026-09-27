import {
	Content as DialogContent,
	Overlay as DialogOverlay,
	Portal as DialogPortal,
	Root as DialogRoot,
	Title as DialogTitle,
} from "@kobalte/core/dialog";
import { Link, useLocation, useNavigate } from "@tanstack/solid-router";
import { cva } from "class-variance-authority";
import { type Accessor, createEffect, createSignal, Show } from "solid-js";
import { FastTrackMark } from "@/components/brand-mark";
import { queryClient } from "@/lib/queryClient";
import { signOut } from "@/lib/session";
import { cn } from "@/lib/utils";
import { logout } from "../api/auth";
import { useApplications } from "../hooks/useApplications";
import { useGoogleStatus } from "../hooks/useGoogle";

const navLinkVariants = cva(
	"flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors",
	{
		variants: {
			active: {
				true: "bg-sidebar-active text-sidebar-active-foreground ring-1 ring-sidebar-active-foreground/20",
				false:
					"text-sidebar-foreground hover:bg-sidebar-hover hover:text-sidebar-foreground-strong",
			},
		},
		defaultVariants: { active: false },
	},
);

const sidebarBadgeVariants = cva(
	"ml-auto rounded-full px-1.5 py-0.5 font-mono text-2xs tabular-nums",
	{
		variants: {
			active: {
				true: "bg-sidebar-badge-active text-sidebar-active-foreground",
				false: "bg-sidebar-badge text-sidebar-foreground",
			},
		},
		defaultVariants: { active: false },
	},
);

interface SidebarProps {
	mobileOpen?: boolean;
	onMobileClose?: () => void;
	restoreFocusTo?: () => HTMLElement | undefined;
}

export default function Sidebar(props: SidebarProps) {
	const [expanded, setExpanded] = createSignal(true);
	const location = useLocation();
	const navigate = useNavigate();

	const isOverviewActive = () => location().pathname === "/overview";
	const isJobsActive = () => location().pathname === "/jobs";
	const isCompaniesActive = () => location().pathname.startsWith("/companies");
	const isApplicationsActive = () => location()?.pathname === "/applications";
	const isInsightsActive = () => location().pathname === "/insights";
	const isSettingsActive = () => location().pathname.startsWith("/settings");
	const isCVTemplatesActive = () => location().pathname === "/cv-templates";

	const appsQuery = useApplications();
	const googleStatus = useGoogleStatus();

	const handleLogout = () =>
		signOut(logout, () => {
			queryClient.clear();
			navigate({ to: "/login" });
		});

	createEffect(() => {
		location().pathname;
		props.onMobileClose?.();
	});

	const links = {
		isOverviewActive,
		isJobsActive,
		isCompaniesActive,
		isApplicationsActive,
		isInsightsActive,
		isSettingsActive,
		isCVTemplatesActive,
		appsQuery,
		googleStatus,
		handleLogout,
	};

	return (
		<>
			<aside
				class={cn(
					"hidden h-screen shrink-0 flex-col overflow-hidden border-r border-sidebar-border bg-sidebar transition-[width] duration-300 md:flex",
					expanded() ? "w-[var(--sidebar-w,13.75rem)]" : "w-14",
				)}
			>
				<SidebarBody
					{...links}
					showLabels={expanded}
					expanded={expanded}
					onCollapseToggle={() => setExpanded((e) => !e)}
				/>
			</aside>

			<DialogRoot
				open={props.mobileOpen ?? false}
				onOpenChange={(open) => {
					if (!open) props.onMobileClose?.();
				}}
				modal
			>
				<DialogPortal>
					<DialogOverlay class="fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px] md:hidden data-[expanded]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:fade-in-0" />
					<DialogContent
						class="fixed inset-y-0 left-0 z-50 flex h-screen w-[var(--sidebar-w,13.75rem)] flex-col overflow-hidden border-r border-sidebar-border bg-sidebar shadow-2xl md:hidden data-[expanded]:animate-in data-[closed]:animate-out data-[expanded]:slide-in-from-left-full data-[closed]:slide-out-to-left-full data-[expanded]:duration-300 data-[closed]:duration-200"
						onCloseAutoFocus={() => props.restoreFocusTo?.()?.focus()}
					>
						<DialogTitle class="sr-only">Navigation</DialogTitle>
						<SidebarBody {...links} showLabels={() => true} />
					</DialogContent>
				</DialogPortal>
			</DialogRoot>
		</>
	);
}

interface SidebarBodyProps {
	showLabels: Accessor<boolean>;
	expanded?: Accessor<boolean>;
	onCollapseToggle?: () => void;
	isOverviewActive: () => boolean;
	isJobsActive: () => boolean;
	isCompaniesActive: () => boolean;
	isApplicationsActive: () => boolean;
	isInsightsActive: () => boolean;
	isSettingsActive: () => boolean;
	isCVTemplatesActive: () => boolean;
	appsQuery: ReturnType<typeof useApplications>;
	googleStatus: ReturnType<typeof useGoogleStatus>;
	handleLogout: () => void;
}

function SidebarBody(props: SidebarBodyProps) {
	return (
		<>
			<div class="flex h-14 shrink-0 items-center gap-2.5 border-b border-sidebar-border px-4">
				<div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-sidebar-active text-sidebar-active-foreground">
					<FastTrackMark size={15} />
				</div>
				<Show when={props.showLabels()}>
					<span class="whitespace-nowrap text-sm font-semibold tracking-tight text-sidebar-foreground-strong">
						FastTrack
					</span>
				</Show>
			</div>

			<nav class="flex flex-1 flex-col gap-0.5 overflow-y-auto px-2 py-3">
				<Show when={props.showLabels()}>
					<span class="px-2.5 pb-1 pt-2.5 text-2xs font-semibold uppercase tracking-wider text-sidebar-foreground">
						Main
					</span>
				</Show>

				<Link
					to="/overview"
					title="Overview"
					class={navLinkVariants({ active: props.isOverviewActive() })}
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
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Overview</span>
					</Show>
				</Link>

				<Link
					to="/jobs"
					title="Jobs"
					class={navLinkVariants({ active: props.isJobsActive() })}
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
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Jobs</span>
					</Show>
				</Link>

				<Link
					to="/applications"
					search={{ status: undefined }}
					title="Applications"
					class={navLinkVariants({ active: props.isApplicationsActive() })}
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
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Applications</span>
						<Show when={(props.appsQuery.data?.length ?? 0) > 0}>
							<span
								class={sidebarBadgeVariants({
									active: props.isApplicationsActive(),
								})}
							>
								{props.appsQuery.data?.length}
							</span>
						</Show>
					</Show>
				</Link>

				<Link
					to="/companies"
					title="Companies"
					class={navLinkVariants({ active: props.isCompaniesActive() })}
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
						<path d="M3 21h18" />
						<path d="M5 21V7l7-4 7 4v14" />
						<path d="M9 9h1" />
						<path d="M14 9h1" />
						<path d="M9 13h1" />
						<path d="M14 13h1" />
						<path d="M9 21v-4h6v4" />
					</svg>
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Companies</span>
					</Show>
				</Link>

				<Show when={props.googleStatus.data?.connected}>
					<Link
						to="/cv-templates"
						title="CVs"
						class={navLinkVariants({ active: props.isCVTemplatesActive() })}
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
						<Show when={props.showLabels()}>
							<span class="whitespace-nowrap">CVs</span>
						</Show>
					</Link>
				</Show>
				<Link
					to="/insights"
					title="Insights"
					class={navLinkVariants({ active: props.isInsightsActive() })}
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
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Insights</span>
					</Show>
				</Link>

				<Link
					to="/settings"
					title="Settings"
					class={navLinkVariants({ active: props.isSettingsActive() })}
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
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Settings</span>
					</Show>
				</Link>
			</nav>

			<div class="flex shrink-0 flex-col gap-0.5 border-t border-sidebar-border px-2 py-2">
				<button
					type="button"
					onClick={() => props.handleLogout()}
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
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Sign out</span>
					</Show>
				</button>
				<Show when={props.onCollapseToggle}>
					<button
						type="button"
						onClick={() => props.onCollapseToggle?.()}
						title={props.expanded?.() ? "Collapse sidebar" : "Expand sidebar"}
						aria-label={
							props.expanded?.() ? "Collapse sidebar" : "Expand sidebar"
						}
						class="hidden w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-hover hover:text-sidebar-foreground-strong md:flex"
					>
						<svg
							aria-hidden="true"
							class="shrink-0 transition-transform duration-300"
							style={{
								transform: props.expanded?.()
									? "rotate(0deg)"
									: "rotate(180deg)",
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
						<Show when={props.showLabels()}>
							<span class="whitespace-nowrap">Collapse</span>
						</Show>
					</button>
				</Show>
			</div>
		</>
	);
}
