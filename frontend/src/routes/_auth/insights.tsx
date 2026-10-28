import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, onMount, Show } from "solid-js";
import { ApplicationPipelineChart } from "@/components/insights/ApplicationPipelineChart";
import { FirstFoundByChart } from "@/components/insights/FirstFoundByChart";
import { JobsBySourceChart } from "@/components/insights/JobsBySourceChart";
import { JobsOverTimeChart } from "@/components/insights/JobsOverTimeChart";
import { ScoreAndGateChart } from "@/components/insights/ScoreAndGateChart";
import { SourceQualityChart } from "@/components/insights/SourceQualityChart";
import { PageHeading } from "@/components/PageHeading";
import { registerCharts } from "@/lib/charts";
import {
	applicationStatusesQueryOptions,
	useApplicationStatuses,
} from "../../hooks/useApplicationStatuses";
import {
	applicationsQueryOptions,
	useApplications,
} from "../../hooks/useApplications";
import { jobInsightsQueryOptions, useJobInsights } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";
import type { JobInsights } from "../../types/job";

registerCharts();

const EMPTY_INSIGHTS: JobInsights = {
	total: 0,
	scored: 0,
	daily: [],
	sources: [],
	scoreBands: [0, 0, 0, 0, 0],
	foundBy: [],
};

export const Route = createFileRoute("/_auth/insights")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(jobInsightsQueryOptions),
			queryClient.ensureQueryData(applicationsQueryOptions()),
			queryClient.ensureQueryData(applicationStatusesQueryOptions),
		]),
	component: InsightsPage,
});

function InsightsPage() {
	const insightsQuery = useJobInsights();
	const appsQuery = useApplications();
	const statusesQuery = useApplicationStatuses();

	const insights = () => insightsQuery.data ?? EMPTY_INSIGHTS;
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
			<PageHeading
				title="Insights"
				subtitle="Visualise your scored, open jobs from the last 90 days"
			/>

			<Show when={chartsReady()}>
				<div class="mb-3 grid grid-cols-1 gap-3 lg:grid-cols-[1fr_280px]">
					<JobsOverTimeChart insights={insights()} />
					<JobsBySourceChart insights={insights()} />
				</div>

				<ApplicationPipelineChart
					applications={applications()}
					statuses={statuses()}
				/>

				<ScoreAndGateChart insights={insights()} />

				<div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
					<SourceQualityChart insights={insights()} />
					<FirstFoundByChart insights={insights()} />
				</div>
			</Show>
		</div>
	);
}
