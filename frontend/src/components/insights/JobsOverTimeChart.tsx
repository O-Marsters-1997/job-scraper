import { Chart as ChartJS } from "chart.js";
import { format } from "date-fns";
import { Line } from "solid-chartjs";
import { createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Card } from "@/components/ui/card";
import {
	createThemedMemo,
	hexAlpha,
	lineChartOptions,
	primaryHex,
	useChartCanvasRef,
} from "@/lib/charts";
import { cn } from "@/lib/utils";
import type { JobInsights } from "@/types/job";

type Preset = "7D" | "30D" | "90D" | "All";
const PRESETS: Preset[] = ["7D", "30D", "90D", "All"];
const PRESET_DAYS: Record<Preset, number | null> = {
	"7D": 7,
	"30D": 30,
	"90D": 90,
	All: null,
};

export function JobsOverTimeChart(props: { insights: JobInsights }) {
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
		const { daily } = props.insights;
		const first = daily[0];
		if (!first) return { datasets: [] };

		const counts = new Map(daily.map((d) => [d.day, d.count]));
		const today = new Date();
		today.setHours(0, 0, 0, 0);
		const chartData: { x: number; y: number }[] = [];
		const cursor = new Date(`${first.day}T00:00:00`);
		while (cursor <= today) {
			const key = format(cursor, "yyyy-MM-dd");
			chartData.push({
				x: new Date(`${key}T12:00:00Z`).getTime(),
				y: counts.get(key) ?? 0,
			});
			cursor.setDate(cursor.getDate() + 1);
		}

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

	const total = () => props.insights.total;
	const hasJobs = () => total() > 0;
	const setCanvas = useChartCanvasRef(
		() =>
			`Line chart of jobs discovered per day, ${total()} job${total() === 1 ? "" : "s"} total`,
	);

	return (
		<Card>
			<div class="flex items-start justify-between border-b border-border px-5 py-4">
				<div>
					<h3 class="text-sm font-semibold text-foreground">Jobs discovered</h3>
					<p class="mt-0.5 text-xs text-faint">Last 90 days</p>
				</div>

				<div class="flex items-center gap-1">
					<div class="flex rounded-md border border-border">
						<For each={PRESETS}>
							{(p) => (
								<button
									type="button"
									onClick={() => applyPreset(p)}
									aria-pressed={activePreset() === p}
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
						<Icon name="history" size={13} />
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
