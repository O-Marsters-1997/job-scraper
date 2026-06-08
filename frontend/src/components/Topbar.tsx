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
	"/settings/statuses": "Statuses",
};

export default function Topbar() {
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

	const pageLabel = () => PAGE_LABELS[location().pathname] ?? null;

	return (
		<header class="flex h-14 shrink-0 items-center justify-between border-b border-border bg-surface px-6">
			<nav aria-label="Breadcrumb" class="flex items-center gap-1.5 text-sm">
				<span class="text-faint">Job Scraper</span>
				<span class="text-border-strong">/</span>
				<Show
					when={jobDetailId()}
					fallback={
						<span class="font-semibold text-foreground">
							{pageLabel() ?? "Job Scraper"}
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
