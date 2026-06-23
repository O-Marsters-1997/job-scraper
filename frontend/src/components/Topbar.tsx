import { Link, useLocation } from "@tanstack/solid-router";
import { Show } from "solid-js";
import { jobsQueryOptions } from "../hooks/useJobs";
import { queryClient } from "../lib/queryClient";
import type { Job } from "../types/job";
import SettingsPanel from "./SettingsPanel";

const PAGE_LABELS: Record<string, string> = {
	"/overview": "Overview",
	"/jobs": "Jobs",
	"/applications": "Applications",
	"/insights": "Insights",
	"/cv-templates": "CV Templates",
	"/settings/statuses": "Statuses",
	"/settings/searches": "Searches",
	"/settings/integrations": "Integrations",
};

// Fallback so new routes get a sensible breadcrumb instead of duplicating the
// product name: title-case the last path segment ("/settings/foo" -> "Foo").
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
}

export default function Topbar(props: TopbarProps) {
	const location = useLocation();

	const jobDetailId = () => {
		const m = location().pathname.match(/^\/jobs\/([^/]+)$/);
		return m ? m[1] : null;
	};

	const jobDetailTitle = () => {
		const id = jobDetailId();
		if (!id) return null;
		const jobs = queryClient.getQueryData<Job[]>(jobsQueryOptions.queryKey);
		return jobs?.find((j) => j.ID === id)?.Title ?? null;
	};

	const pageLabel = () =>
		PAGE_LABELS[location().pathname] ?? deriveLabel(location().pathname);

	return (
		<header class="flex h-14 shrink-0 items-center justify-between border-b border-border bg-surface px-6">
			<nav aria-label="Breadcrumb" class="flex items-center gap-1.5 text-sm">
				<button
					type="button"
					onClick={() => props.onMenuClick?.()}
					aria-label="Open menu"
					class="-ml-1 mr-1 inline-flex h-8 w-8 items-center justify-center rounded-md text-muted transition-colors hover:bg-surface-muted hover:text-foreground md:hidden"
				>
					<svg
						aria-hidden="true"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<line x1="3" y1="6" x2="21" y2="6" />
						<line x1="3" y1="12" x2="21" y2="12" />
						<line x1="3" y1="18" x2="21" y2="18" />
					</svg>
				</button>
				<span class="text-faint">FastTrack</span>
				<span class="text-border-strong">/</span>
				<Show
					when={jobDetailId()}
					fallback={
						<span class="font-semibold text-foreground">
							{pageLabel() ?? "FastTrack"}
						</span>
					}
				>
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
				</Show>
			</nav>
			<SettingsPanel />
		</header>
	);
}
