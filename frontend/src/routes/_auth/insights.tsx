import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, onMount } from "solid-js";
import { ApplicationPipelineChart } from "@/components/insights/ApplicationPipelineChart";
import { JobsBySourceChart } from "@/components/insights/JobsBySourceChart";
import { JobsOverTimeChart } from "@/components/insights/JobsOverTimeChart";
import { ScoreAndGateChart } from "@/components/insights/ScoreAndGateChart";
import { SkillGapChart } from "@/components/insights/SkillGapChart";
import { SourceQualityChart } from "@/components/insights/SourceQualityChart";
import { registerCharts } from "@/lib/charts";
import {
	applicationStatusesQueryOptions,
	useApplicationStatuses,
} from "../../hooks/useApplicationStatuses";
import {
	applicationsQueryOptions,
	useApplications,
} from "../../hooks/useApplications";
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";

registerCharts();

export const Route = createFileRoute("/_auth/insights")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(jobsQueryOptions),
			queryClient.ensureQueryData(applicationsQueryOptions()),
			queryClient.ensureQueryData(applicationStatusesQueryOptions),
		]),
	component: InsightsPage,
});

function InsightsPage() {
	const jobsQuery = useJobs();
	const appsQuery = useApplications();
	const statusesQuery = useApplicationStatuses();

	const jobs = () => jobsQuery.data ?? [];
	const applications = () => appsQuery.data ?? [];
	const statuses = () => statusesQuery.data ?? [];

	// Defer chart.js init until after a layout pass. During a client route
	// transition the canvas can mount before its ownerDocument has a live
	// defaultView, and chart.js throws reading getComputedStyle on it. A single
	// rAF guarantees the canvas is connected and laid out before charts render.
	const [chartsReady, setChartsReady] = createSignal(false);
	onMount(() => requestAnimationFrame(() => setChartsReady(true)));

	return (
		<div class="px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Insights
				</h1>
				<p class="mt-0.5 text-xs text-faint">Visualise your job search</p>
			</div>

			<div class="mb-3 grid grid-cols-1 gap-3 lg:grid-cols-[1fr_280px]">
				<JobsOverTimeChart jobs={jobs()} chartsReady={chartsReady()} />
				<JobsBySourceChart jobs={jobs()} chartsReady={chartsReady()} />
			</div>

			<ApplicationPipelineChart
				applications={applications()}
				statuses={statuses()}
				chartsReady={chartsReady()}
			/>

			<ScoreAndGateChart jobs={jobs()} chartsReady={chartsReady()} />

			<SourceQualityChart jobs={jobs()} chartsReady={chartsReady()} />

			<SkillGapChart jobs={jobs()} chartsReady={chartsReady()} />
		</div>
	);
}
