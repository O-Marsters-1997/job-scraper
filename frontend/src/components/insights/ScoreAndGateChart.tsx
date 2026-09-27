import { Bar } from "solid-chartjs";
import { createMemo, Show } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	createThemedMemo,
	hexAlpha,
	primaryHex,
	useChartCanvasRef,
	verticalBarOptions,
} from "@/lib/charts";
import type { Job } from "@/types/job";

const SCORE_BANDS = ["0–19", "20–39", "40–59", "60–79", "80–100"];

export function ScoreAndGateChart(props: { jobs: Job[] }) {
	const gateStats = createMemo(() => {
		let scored = 0;
		let unscored = 0;
		for (const job of props.jobs) {
			if (job.SuitabilityScore != null) scored++;
			else unscored++;
		}
		return { scored, unscored };
	});

	const hasScores = () => gateStats().scored > 0;

	const scoreDistData = createThemedMemo(() => {
		const bands = [0, 0, 0, 0, 0];
		for (const job of props.jobs) {
			if (job.SuitabilityScore != null) {
				const idx = Math.min(Math.floor(job.SuitabilityScore / 20), 4);
				bands[idx] = (bands[idx] ?? 0) + 1;
			}
		}
		const primary = primaryHex();
		return {
			labels: SCORE_BANDS,
			datasets: [
				{
					data: bands,
					backgroundColor: hexAlpha(primary, "26"),
					borderColor: primary,
					borderWidth: 1.5,
					borderRadius: 3,
				},
			],
		};
	});

	const options = createThemedMemo(verticalBarOptions);

	const setCanvas = useChartCanvasRef(
		() =>
			`Bar chart of suitability score distribution, ${gateStats().scored} scored job${gateStats().scored === 1 ? "" : "s"}`,
	);

	return (
		<Card class="mb-3 mt-3">
			<div class="border-b border-border px-5 py-4">
				<h3 class="text-sm font-semibold text-foreground">
					Suitability scores
				</h3>
				<p class="mt-0.5 text-xs text-faint">
					How the suitability scorer has rated your feed so far
				</p>
			</div>
			<div class="grid grid-cols-1 lg:grid-cols-[1fr_180px]">
				<div class="px-5 py-4 lg:border-r lg:border-border">
					<p class="mb-2 text-xs text-faint">Suitability score distribution</p>
					<Show
						when={hasScores()}
						fallback={<p class="text-sm text-faint">No scored jobs yet.</p>}
					>
						<div class="relative" style={{ height: "140px" }}>
							<Bar ref={setCanvas} data={scoreDistData()} options={options()} />
						</div>
					</Show>
				</div>
				<div class="divide-y divide-border">
					<div class="flex items-center justify-between px-5 py-3.5">
						<div>
							<p class="text-xs font-medium text-muted">Scored</p>
							<p class="mt-0.5 text-xs text-faint">LLM evaluated</p>
						</div>
						<p class="font-mono text-lg font-medium tabular-nums text-foreground">
							{gateStats().scored}
						</p>
					</div>
					<div class="flex items-center justify-between px-5 py-3.5">
						<div>
							<p class="text-xs font-medium text-muted">Pending</p>
							<p class="mt-0.5 text-xs text-faint">Not yet scored</p>
						</div>
						<p class="font-mono text-lg font-medium tabular-nums text-foreground">
							{gateStats().unscored}
						</p>
					</div>
				</div>
			</div>
		</Card>
	);
}
