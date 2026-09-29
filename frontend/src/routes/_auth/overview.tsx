import { createFileRoute } from "@tanstack/solid-router";
import { createMemo } from "solid-js";
import {
	applicationStats,
	jobStats,
	pipelineSegments,
	RECENT_LIMIT,
	recentJobs,
} from "@/lib/overview";
import {
	applicationStatusesQueryOptions,
	useApplicationStatuses,
} from "../../hooks/useApplicationStatuses";
import {
	applicationsQueryOptions,
	useApplications,
} from "../../hooks/useApplications";
import { allJobsQueryOptions, useAllJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";
import { PipelineCard } from "./-overview/PipelineCard";
import { RecentApplicationsCard } from "./-overview/RecentApplicationsCard";
import { RecentJobsCard } from "./-overview/RecentJobsCard";
import { StatCards } from "./-overview/StatCards";

export const Route = createFileRoute("/_auth/overview")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(allJobsQueryOptions),
			queryClient.ensureQueryData(applicationsQueryOptions()),
			queryClient.ensureQueryData(applicationStatusesQueryOptions),
		]),
	component: OverviewPage,
});

function OverviewPage() {
	const jobsQuery = useAllJobs();
	const appsQuery = useApplications();
	const statusesQuery = useApplicationStatuses();

	const jobs = () => jobsQuery.data ?? [];
	const applications = () => appsQuery.data ?? [];
	const statuses = () => statusesQuery.data ?? [];

	const jobSummary = createMemo(() => jobStats(jobs()));
	const appSummary = createMemo(() =>
		applicationStats(applications(), statuses()),
	);
	const segments = createMemo(() =>
		pipelineSegments(applications(), statuses()),
	);
	const recent = createMemo(() => recentJobs(jobs()));

	return (
		<div class="px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Overview
				</h1>
				<p class="mt-0.5 text-xs text-faint">Your job search at a glance</p>
			</div>

			<StatCards jobs={jobSummary()} apps={appSummary()} />
			<PipelineCard segments={segments()} />
			<div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_300px]">
				<RecentJobsCard jobs={recent()} />
				<RecentApplicationsCard
					applications={applications().slice(0, RECENT_LIMIT)}
				/>
			</div>
		</div>
	);
}
