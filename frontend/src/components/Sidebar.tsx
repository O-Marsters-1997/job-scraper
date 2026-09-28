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
import { Icon } from "@/components/Icon";
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
					<Icon name="dashboard" class="shrink-0" />
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Overview</span>
					</Show>
				</Link>

				<Link
					to="/jobs"
					title="Jobs"
					class={navLinkVariants({ active: props.isJobsActive() })}
				>
					<Icon name="suitcase" class="shrink-0" />
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
					<Icon name="fileText" class="shrink-0" />
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
					<Icon name="building" class="shrink-0" />
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
						<Icon name="fileLines" class="shrink-0" />
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
					<Icon name="barChart" class="shrink-0" />
					<Show when={props.showLabels()}>
						<span class="whitespace-nowrap">Insights</span>
					</Show>
				</Link>

				<Link
					to="/settings"
					title="Settings"
					class={navLinkVariants({ active: props.isSettingsActive() })}
				>
					<Icon name="settings" class="shrink-0" />
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
					<Icon name="logOut" class="shrink-0" />
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
						<Icon
							name="chevronLeft"
							class="shrink-0 transition-transform duration-300"
							style={{
								transform: props.expanded?.()
									? "rotate(0deg)"
									: "rotate(180deg)",
							}}
						/>
						<Show when={props.showLabels()}>
							<span class="whitespace-nowrap">Collapse</span>
						</Show>
					</button>
				</Show>
			</div>
		</>
	);
}
