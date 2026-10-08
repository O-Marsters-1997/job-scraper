import { createFileRoute } from "@tanstack/solid-router";
import { createMemo } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import {
	applicationStats,
	chasesDue,
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
	chasesQueryOptions,
	useApplications,
	useChases,
} from "../../hooks/useApplications";
import { allJobsQueryOptions, useAllJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";
import { ChasesDueCard } from "./-overview/ChasesDueCard";
import { PipelineCard } from "./-overview/PipelineCard";
import { RecentApplicationsCard } from "./-overview/RecentApplicationsCard";
import { RecentJobsCard } from "./-overview/RecentJobsCard";
import { StatCards } from "./-overview/StatCards";

export const Route = createFileRoute("/_auth/overview")({
	loader: () => {
		void queryClient.prefetchQuery(chasesQueryOptions);
		return Promise.all([
			queryClient.ensureQueryData(allJobsQueryOptions),
			queryClient.ensureQueryData(applicationsQueryOptions()),
			queryClient.ensureQueryData(applicationStatusesQueryOptions),
		]);
	},
	component: OverviewPage,
});

function OverviewPage() {
	const jobsQuery = useAllJobs();
	const appsQuery = useApplications();
	const chasesQuery = useChases();
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
	const due = createMemo(() => chasesDue(chasesQuery.data ?? []));
	const recent = createMemo(() => recentJobs(jobs()));

	return (
		<div class="px-7 py-6">
			<PageHeading title="Overview" subtitle="Your job search at a glance" />

			<StatCards jobs={jobSummary()} apps={appSummary()} />
			<PipelineCard segments={segments()} />
			<ChasesDueCard chases={due().slice(0, RECENT_LIMIT)} />
			<div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_300px]">
				<RecentJobsCard jobs={recent()} />
				<RecentApplicationsCard
					applications={applications().slice(0, RECENT_LIMIT)}
				/>
			</div>
		</div>
	);
}
