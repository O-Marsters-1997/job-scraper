import { Bar } from "solid-chartjs";
import { createMemo, Show } from "solid-js";
import { Card } from "@/components/ui/card";
import { hexAlpha, horizontalBarOptions, sourceHex } from "@/lib/charts";
import type { Job } from "@/types/job";

export function SourceQualityChart(props: {
	jobs: Job[];
	chartsReady: boolean;
}) {
	const data = createMemo(() => {
		const scoreSum: Record<string, number> = {};
		const scoreCount: Record<string, number> = {};
		for (const job of props.jobs) {
			if (job.SuitabilityScore != null) {
				scoreSum[job.Source] =
					(scoreSum[job.Source] ?? 0) + job.SuitabilityScore;
				scoreCount[job.Source] = (scoreCount[job.Source] ?? 0) + 1;
			}
		}
		const sources = Object.keys(scoreCount).sort(
			(a, b) =>
				(scoreSum[b] ?? 0) / (scoreCount[b] ?? 1) -
				(scoreSum[a] ?? 0) / (scoreCount[a] ?? 1),
		);
		return {
			labels: sources,
			datasets: [
				{
					data: sources.map((s) =>
						Math.round((scoreSum[s] ?? 0) / (scoreCount[s] ?? 1)),
					),
					backgroundColor: sources.map((s) => hexAlpha(sourceHex(s), "26")),
					borderColor: sources.map((s) => sourceHex(s)),
					borderWidth: 1.5,
					borderRadius: 3,
				},
			],
		};
	});

	const hasScores = () => data().labels.length > 0;

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
					when={hasScores() && props.chartsReady}
					fallback={<p class="text-sm text-faint">No scored jobs yet.</p>}
				>
					<div
						class="relative"
						style={{
							height: `${Math.max(80, data().labels.length * 40)}px`,
						}}
					>
						<Bar data={data()} options={horizontalBarOptions()} />
					</div>
				</Show>
			</div>
		</Card>
	);
}
