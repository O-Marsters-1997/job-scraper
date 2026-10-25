import { Bar } from "solid-chartjs";
import { Show } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	createThemedMemo,
	hexAlpha,
	horizontalBarOptions,
	sourceHex,
	useChartCanvasRef,
} from "@/lib/charts";
import type { JobInsights } from "@/types/job";

export function SourceQualityChart(props: { insights: JobInsights }) {
	const data = createThemedMemo(() => {
		const scored = props.insights.sources
			.filter((s) => s.scored > 0)
			.sort((a, b) => b.avgScore - a.avgScore);
		const sources = scored.map((s) => s.source);
		return {
			labels: sources,
			datasets: [
				{
					data: scored.map((s) => s.avgScore),
					backgroundColor: sources.map((s) => hexAlpha(sourceHex(s), "26")),
					borderColor: sources.map((s) => sourceHex(s)),
					borderWidth: 1.5,
					borderRadius: 3,
				},
			],
		};
	});

	const options = createThemedMemo(horizontalBarOptions);

	const hasScores = () => data().labels.length > 0;
	const setCanvas = useChartCanvasRef(
		() =>
			`Horizontal bar chart of average suitability score by source, ${data().labels.length} source${data().labels.length === 1 ? "" : "s"}`,
	);

	return (
		<Card class="mb-3">
			<div class="border-b border-border px-5 py-4">
				<h3 class="text-sm font-semibold text-foreground">Source quality</h3>
				<p class="mt-0.5 text-xs text-faint">
					Average suitability score per job board
				</p>
			</div>
			<div class="px-5 py-4">
				<Show
					when={hasScores()}
					fallback={<p class="text-sm text-faint">No scored jobs yet.</p>}
				>
					<div
						class="relative"
						style={{
							height: `${Math.max(80, data().labels.length * 40)}px`,
						}}
					>
						<Bar ref={setCanvas} data={data()} options={options()} />
					</div>
				</Show>
			</div>
		</Card>
	);
}
