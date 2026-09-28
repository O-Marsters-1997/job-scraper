import { createQuery } from "@tanstack/solid-query";
import { Link, useLocation } from "@tanstack/solid-router";
import { Match, Switch } from "solid-js";
import { Icon } from "@/components/Icon";
import { jobQueryOptions } from "../hooks/useJobs";
import { findSettingsSection } from "../lib/settingsSections";
import SettingsPanel from "./SettingsPanel";

const PAGE_LABELS: Record<string, string> = {
	"/overview": "Overview",
	"/jobs": "Jobs",
	"/applications": "Applications",
	"/insights": "Insights",
	"/cv-templates": "CV Templates",
};

function deriveLabel(pathname: string): string | null {
	const segment = pathname.split("/").filter(Boolean).pop();
	if (!segment) return null;
	return segment
		.split("-")
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(" ");
}

interface TopbarProps {
	onMenuClick?: () => void;
	menuButtonRef?: (el: HTMLButtonElement) => void;
}

export default function Topbar(props: TopbarProps) {
	const location = useLocation();

	const jobDetailId = () => {
		const m = location().pathname.match(/^\/jobs\/([^/]+)$/);
		return m ? m[1] : null;
	};

	const jobQuery = createQuery(() => ({
		...jobQueryOptions(jobDetailId() ?? ""),
		enabled: jobDetailId() != null,
	}));
	const jobDetailTitle = () => jobQuery.data?.Title ?? null;

	const pageLabel = () =>
		PAGE_LABELS[location().pathname] ?? deriveLabel(location().pathname);

	const settingsSection = () => findSettingsSection(location().pathname);

	return (
		<header class="flex h-14 shrink-0 items-center justify-between border-b border-border bg-surface px-6">
			<nav aria-label="Breadcrumb" class="flex items-center gap-1.5 text-sm">
				<button
					type="button"
					ref={props.menuButtonRef}
					onClick={() => props.onMenuClick?.()}
					aria-label="Open menu"
					class="-ml-1 mr-1 inline-flex h-8 w-8 items-center justify-center rounded-md text-muted transition-colors hover:bg-surface-muted hover:text-foreground md:hidden"
				>
					<Icon name="menu" size={18} />
				</button>
				<span class="text-faint">FastTrack</span>
				<span class="text-border-strong">/</span>
				<Switch
					fallback={
						<span class="font-semibold text-foreground">
							{pageLabel() ?? "FastTrack"}
						</span>
					}
				>
					<Match when={jobDetailId()}>
						<Link
							to="/jobs"
							class="text-faint transition-colors hover:text-foreground"
						>
							Jobs
						</Link>
						<span class="text-border-strong">/</span>
						<span class="font-semibold text-foreground">
							{jobDetailTitle() ?? "Job"}
						</span>
					</Match>
					<Match when={settingsSection()}>
						{(section) => (
							<>
								<Link
									to="/settings"
									class="text-faint transition-colors hover:text-foreground"
								>
									Settings
								</Link>
								<span class="text-border-strong">/</span>
								<span class="font-semibold text-foreground">
									{section().label}
								</span>
							</>
						)}
					</Match>
				</Switch>
			</nav>
			<SettingsPanel />
		</header>
	);
}
