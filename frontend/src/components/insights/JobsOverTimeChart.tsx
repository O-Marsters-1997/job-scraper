import { Chart as ChartJS } from "chart.js";
import { Line } from "solid-chartjs";
import { createSignal, For, Show } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	createThemedMemo,
	hexAlpha,
	lineChartOptions,
	primaryHex,
	useChartCanvasRef,
} from "@/lib/charts";
import { dayKey } from "@/lib/datetime";
import { cn } from "@/lib/utils";
import type { Job } from "@/types/job";

type Preset = "7D" | "30D" | "90D" | "All";
const PRESETS: Preset[] = ["7D", "30D", "90D", "All"];
const PRESET_DAYS: Record<Preset, number | null> = {
	"7D": 7,
	"30D": 30,
	"90D": 90,
	All: null,
};

export function JobsOverTimeChart(props: { jobs: Job[] }) {
	const [activePreset, setActivePreset] = createSignal<Preset>("All");
	let lineCanvas: HTMLCanvasElement | null = null;

	function applyPreset(preset: Preset) {
		setActivePreset(preset);
		const chart = lineCanvas ? ChartJS.getChart(lineCanvas) : null;
		if (!chart) return;
		const days = PRESET_DAYS[preset];
		if (days === null) {
			chart.resetZoom();
			return;
		}
		const max = Date.now();
		const min = max - days * 24 * 60 * 60 * 1000;
		chart.zoomScale("x", { min, max }, "default");
	}

	function resetZoom() {
		setActivePreset("All");
		const chart = lineCanvas ? ChartJS.getChart(lineCanvas) : null;
		chart?.resetZoom();
	}

	const data = createThemedMemo(() => {
		const allJobs = props.jobs;
		if (allJobs.length === 0) return { datasets: [] };

		const timestamps = allJobs.map((j) => new Date(j.ScrapedAt).getTime());
		const earliest = new Date(Math.min(...timestamps));
		earliest.setHours(0, 0, 0, 0);
		const today = new Date();
		today.setHours(0, 0, 0, 0);

		const spine = new Map<string, number>();
		const cursor = new Date(earliest);
		while (cursor <= today) {
			spine.set(dayKey(cursor), 0);
			cursor.setDate(cursor.getDate() + 1);
		}

		for (const job of allJobs) {
			const key = dayKey(job.ScrapedAt);
			if (spine.has(key)) spine.set(key, (spine.get(key) ?? 0) + 1);
		}

		const chartData = [...spine.entries()].map(([k, count]) => ({
			x: new Date(`${k}T12:00:00Z`).getTime(),
			y: count,
		}));

		return {
			datasets: [
				{
					data: chartData,
					borderColor: primaryHex(),
					backgroundColor: hexAlpha(primaryHex(), "1a"),
					fill: true,
					tension: 0.35,
					pointRadius: 2,
					pointHoverRadius: 4,
				},
			],
		};
	});

	const options = createThemedMemo(lineChartOptions);

	const hasJobs = () => props.jobs.length > 0;
	const setCanvas = useChartCanvasRef(
		() =>
			`Line chart of jobs discovered per day, ${props.jobs.length} job${props.jobs.length === 1 ? "" : "s"} total`,
	);

	return (
		<Card>
			<div class="flex items-start justify-between border-b border-border px-5 py-4">
				<div>
					<h3 class="text-sm font-semibold text-foreground">Jobs discovered</h3>
					<p class="mt-0.5 text-xs text-faint">All time</p>
				</div>

				<div class="flex items-center gap-1">
					<div class="flex rounded-md border border-border">
						<For each={PRESETS}>
							{(p) => (
								<button
									type="button"
									onClick={() => applyPreset(p)}
									class={cn(
										"px-2.5 py-1 text-xs font-medium transition-colors first:rounded-l-[5px] last:rounded-r-[5px] not-last:border-r not-last:border-border",
										activePreset() === p
											? "bg-primary/10 text-primary"
											: "text-muted hover:bg-surface-muted hover:text-foreground",
									)}
								>
									{p}
								</button>
							)}
						</For>
					</div>
					<button
						type="button"
						onClick={resetZoom}
						title="Reset zoom"
						class="ml-1 flex size-7 items-center justify-center rounded-md border border-border text-muted transition-colors hover:bg-surface-muted hover:text-foreground"
					>
						<svg
							aria-hidden="true"
							width="13"
							height="13"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
							<path d="M3 3v5h5" />
						</svg>
					</button>
				</div>
			</div>

			<div class="relative px-5 pt-4" style={{ height: "220px" }}>
				<Show
					when={hasJobs()}
					fallback={<p class="text-sm text-faint">No jobs scraped yet.</p>}
				>
					<Line
						ref={(c: HTMLCanvasElement | null) => {
							lineCanvas = c;
							setCanvas(c);
						}}
						data={data()}
						options={options()}
					/>
				</Show>
			</div>
			<p class="px-5 pb-3 pt-1.5 text-2xs text-faint">
				Drag to zoom · Scroll to zoom · Shift + drag to pan
			</p>
		</Card>
	);
}
