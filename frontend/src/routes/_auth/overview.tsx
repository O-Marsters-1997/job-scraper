import { createFileRoute } from "@tanstack/solid-router";
import { createMemo } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import {
	applicationStats,
	chasesDue,
	pipelineSegments,
	RECENT_LIMIT,
} from "@/lib/overview";
import { aiPrefsQueryOptions, useAiPrefs } from "../../hooks/useAiPrefs";
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
import { jobOverviewQueryOptions, useJobOverview } from "../../hooks/useJobs";
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
			queryClient
				.ensureQueryData(aiPrefsQueryOptions)
				.then((prefs) =>
					queryClient.ensureQueryData(
						jobOverviewQueryOptions(prefs.scoringEnabled),
					),
				),
			queryClient.ensureQueryData(applicationsQueryOptions()),
			queryClient.ensureQueryData(applicationStatusesQueryOptions),
		]);
	},
	component: OverviewPage,
});

function OverviewPage() {
	const aiPrefs = useAiPrefs();
	const ranked = () => aiPrefs.data?.scoringEnabled ?? true;
	const overview = useJobOverview(ranked);
	const appsQuery = useApplications();
	const chasesQuery = useChases();
	const statusesQuery = useApplicationStatuses();

	const applications = () => appsQuery.data ?? [];
	const statuses = () => statusesQuery.data ?? [];

	const appSummary = createMemo(() =>
		applicationStats(applications(), statuses()),
	);
	const segments = createMemo(() =>
		pipelineSegments(applications(), statuses()),
	);
	const due = createMemo(() => chasesDue(chasesQuery.data ?? []));
	const jobSummary = () =>
		overview.data ?? { total: 0, sources: 0, newToday: 0, todaySources: 0 };

	return (
		<div class="px-7 py-6">
			<PageHeading title="Overview" subtitle="Your job search at a glance" />

			<StatCards jobs={jobSummary()} apps={appSummary()} />
			<PipelineCard segments={segments()} />
			<ChasesDueCard chases={due().slice(0, RECENT_LIMIT)} />
			<div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_300px]">
				<RecentJobsCard jobs={overview.data?.top ?? []} ranked={ranked()} />
				<RecentApplicationsCard
					applications={applications().slice(0, RECENT_LIMIT)}
				/>
			</div>
		</div>
	);
}
