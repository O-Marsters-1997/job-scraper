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
import { firstFoundBy } from "@/lib/insights";
import type { Job } from "@/types/job";

export function FirstFoundByChart(props: { jobs: Job[] }) {
	const data = createThemedMemo(() => {
		const counts: Record<string, number> = {};
		for (const job of props.jobs) {
			if (job.Band !== "good" && job.Band !== "great") continue;
			const who = firstFoundBy(job);
			counts[who] = (counts[who] ?? 0) + 1;
		}
		const finders = Object.keys(counts).sort(
			(a, b) => (counts[b] ?? 0) - (counts[a] ?? 0),
		);
		return {
			labels: finders,
			datasets: [
				{
					data: finders.map((f) => counts[f] ?? 0),
					backgroundColor: finders.map((f) => hexAlpha(sourceHex(f), "26")),
					borderColor: finders.map((f) => sourceHex(f)),
					borderWidth: 1.5,
					borderRadius: 3,
				},
			],
		};
	});

	const options = createThemedMemo(horizontalBarOptions);

	const hasJobs = () => data().labels.length > 0;
	const setCanvas = useChartCanvasRef(
		() =>
			`Horizontal bar chart of good and great jobs by who found them first, ${data().labels.length} finder${data().labels.length === 1 ? "" : "s"}`,
	);

	return (
		<Card class="mb-3">
			<div class="border-b border-border px-5 py-4">
				<h3 class="text-sm font-semibold text-foreground">First found by</h3>
				<p class="mt-0.5 text-xs text-faint">
					Good and great jobs, credited to whoever surfaced them first
				</p>
			</div>
			<div class="px-5 py-4">
				<Show
					when={hasJobs()}
					fallback={
						<p class="text-sm text-faint">No good or great jobs yet.</p>
					}
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
