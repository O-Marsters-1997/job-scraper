import { Doughnut } from "solid-chartjs";
import { Show } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	createThemedMemo,
	donutChartOptions,
	hexAlpha,
	sourceHex,
	useChartCanvasRef,
} from "@/lib/charts";
import type { JobInsights } from "@/types/job";

export function JobsBySourceChart(props: { insights: JobInsights }) {
	const data = createThemedMemo(() => {
		const entries = props.insights.sources.map(
			(s) => [s.source, s.count] as const,
		);
		return {
			labels: entries.map(([s]) => s),
			datasets: [
				{
					data: entries.map(([, n]) => n),
					backgroundColor: entries.map(([s]) => hexAlpha(sourceHex(s), "26")),
					borderColor: entries.map(([s]) => sourceHex(s)),
					borderWidth: 1.5,
					hoverOffset: 4,
				},
			],
		};
	});

	const options = createThemedMemo(donutChartOptions);

	const count = () => props.insights.total;
	const hasJobs = () => count() > 0;
	const sourceCount = () => props.insights.sources.length;
	const setCanvas = useChartCanvasRef(
		() =>
			`Doughnut chart of jobs by source, ${count()} job${count() === 1 ? "" : "s"} across ${sourceCount()} source${sourceCount() === 1 ? "" : "s"}`,
	);

	return (
		<Card>
			<div class="border-b border-border px-5 py-4">
				<h3 class="text-sm font-semibold text-foreground">By source</h3>
				<p class="mt-0.5 text-xs text-faint">
					{count()} job{count() === 1 ? "" : "s"} total
				</p>
			</div>
			<div class="relative px-5 py-4" style={{ height: "220px" }}>
				<Show
					when={hasJobs()}
					fallback={<p class="text-sm text-faint">No jobs scraped yet.</p>}
				>
					<Doughnut ref={setCanvas} data={data()} options={options()} />
				</Show>
			</div>
		</Card>
	);
}
