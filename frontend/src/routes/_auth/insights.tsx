import { createFileRoute } from "@tanstack/solid-router";
import { Chart as ChartJS } from "chart.js";
import {
	createEffect,
	createMemo,
	createSignal,
	For,
	onMount,
	Show,
} from "solid-js";
import { Bar, Doughnut, Line } from "solid-chartjs";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import {
	donutChartOptions,
	hexAlpha,
	lineChartOptions,
	registerCharts,
	sourceHex,
	stackedBarOptions,
} from "@/lib/charts";
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

type Preset = "7D" | "30D" | "90D" | "All";
const PRESETS: Preset[] = ["7D", "30D", "90D", "All"];
const PRESET_DAYS: Record<Preset, number | null> = {
	"7D": 7,
	"30D": 30,
	"90D": 90,
	All: null,
};

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

	// ── C1: range preset state + canvas ref ───────────────────────────────────
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

	// ── A1: Jobs discovered — full date-range time series ─────────────────────
	const jobsOverTimeData = createMemo(() => {
		const allJobs = jobs();
		if (allJobs.length === 0) return { datasets: [] };

		const timestamps = allJobs.map((j) => new Date(j.ScrapedAt).getTime());
		const earliest = new Date(Math.min(...timestamps));
		earliest.setHours(0, 0, 0, 0);
		const today = new Date();
		today.setHours(0, 0, 0, 0);

		// Build daily spine from earliest job → today
		const spine = new Map<string, number>();
		const cursor = new Date(earliest);
		while (cursor <= today) {
			spine.set(cursor.toISOString().slice(0, 10), 0);
			cursor.setDate(cursor.getDate() + 1);
		}

		for (const job of allJobs) {
			const key = new Date(job.ScrapedAt).toISOString().slice(0, 10);
			if (spine.has(key)) spine.set(key, (spine.get(key) ?? 0) + 1);
		}

		const data = [...spine.entries()].map(([k, count]) => ({
			x: new Date(`${k}T12:00:00Z`).getTime(),
			y: count,
		}));

		return {
			datasets: [
				{
					data,
					borderColor: "#0f9d92",
					backgroundColor: hexAlpha("#0f9d92", "1a"),
					fill: true,
					tension: 0.35,
					pointRadius: 2,
					pointHoverRadius: 4,
				},
			],
		};
	});

	// ── A2: Jobs by source ────────────────────────────────────────────────────
	const jobsBySourceData = createMemo(() => {
		const counts: Record<string, number> = {};
		for (const job of jobs()) {
			counts[job.Source] = (counts[job.Source] ?? 0) + 1;
		}
		const entries = Object.entries(counts).sort((a, b) => b[1] - a[1]);
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

	// ── C2: Pipeline — multiselect state ─────────────────────────────────────
	const [selectedStatuses, setSelectedStatuses] = createSignal(
		new Set<string>(),
	);

	// Seed selection when statuses first load (guaranteed by loader, so this
	// fires once synchronously on mount).
	createEffect(() => {
		const ids = statuses().map((s) => s.ID);
		if (ids.length > 0 && selectedStatuses().size === 0) {
			setSelectedStatuses(new Set(ids));
		}
	});

	function toggleStatus(id: string) {
		setSelectedStatuses((prev) => {
			const next = new Set(prev);
			if (next.has(id)) next.delete(id);
			else next.add(id);
			return next;
		});
	}

	// ── A3: Pipeline chart data ───────────────────────────────────────────────
	const appsByStatus = createMemo(() => {
		const map: Record<string, number> = {};
		for (const app of applications()) {
			map[app.StatusID] = (map[app.StatusID] ?? 0) + 1;
		}
		return map;
	});

	const pipelineData = createMemo(() => ({
		labels: [""],
		datasets: statuses()
			.filter((s) => selectedStatuses().has(s.ID))
			.map((s) => ({
				label: s.Name,
				data: [appsByStatus()[s.ID] ?? 0],
				backgroundColor: hexAlpha(s.Colour, "26"),
				borderColor: s.Colour,
				borderWidth: 1.5,
				borderRadius: 3,
			})),
	}));

	const totalApps = () => applications().length;
	const hasJobs = () => jobs().length > 0;

	return (
		<div class="px-7 py-6">
			{/* Page header */}
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Insights
				</h1>
				<p class="mt-0.5 text-xs text-faint">Visualise your job search</p>
			</div>

			{/* Top row: Jobs over time + Jobs by source */}
			<div class="mb-3 grid grid-cols-1 gap-3 lg:grid-cols-[1fr_280px]">
				{/* C1 — Jobs discovered with range presets + zoom */}
				<Card>
					<div class="flex items-start justify-between border-b border-border px-5 py-4">
						<div>
							<h3 class="text-sm font-semibold text-foreground">
								Jobs discovered
							</h3>
							<p class="mt-0.5 text-xs text-faint">All time</p>
						</div>

						{/* Preset buttons */}
						<div class="flex items-center gap-1">
							<div class="flex rounded-md border border-border">
								<For each={PRESETS}>
									{(p) => (
										<button
											type="button"
											onClick={() => applyPreset(p)}
											class={cn(
												"px-2.5 py-1 text-[11px] font-medium transition-colors first:rounded-l-[5px] last:rounded-r-[5px] not-last:border-r not-last:border-border",
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
							{/* Reset */}
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
							when={hasJobs() && chartsReady()}
							fallback={<p class="text-sm text-faint">No jobs scraped yet.</p>}
						>
							<Line
								ref={(c: HTMLCanvasElement | null) => {
									lineCanvas = c;
								}}
								data={jobsOverTimeData()}
								options={lineChartOptions}
							/>
						</Show>
					</div>
					<p class="px-5 pb-3 pt-1.5 text-[10px] text-faint">
						Drag to zoom · Scroll to zoom · Shift + drag to pan
					</p>
				</Card>

				{/* A2 — Jobs by source */}
				<Card>
					<div class="border-b border-border px-5 py-4">
						<h3 class="text-sm font-semibold text-foreground">By source</h3>
						<p class="mt-0.5 text-xs text-faint">
							{jobs().length} job{jobs().length === 1 ? "" : "s"} total
						</p>
					</div>
					<div class="relative px-5 py-4" style={{ height: "220px" }}>
						<Show
							when={hasJobs() && chartsReady()}
							fallback={<p class="text-sm text-faint">No jobs scraped yet.</p>}
						>
							<Doughnut data={jobsBySourceData()} options={donutChartOptions} />
						</Show>
					</div>
				</Card>
			</div>

			{/* C2 — Application pipeline with custom multiselect legend */}
			<Card>
				<div class="border-b border-border px-5 py-4">
					<h3 class="text-sm font-semibold text-foreground">
						Application pipeline
					</h3>
					<p class="mt-0.5 text-xs text-faint">
						{totalApps()} application{totalApps() === 1 ? "" : "s"} across{" "}
						{statuses().length} stage{statuses().length === 1 ? "" : "s"}
					</p>
				</div>
				<div class="px-5 py-4">
					<Show
						when={totalApps() > 0}
						fallback={<p class="text-sm text-faint">No applications yet.</p>}
					>
						{/* Multiselect chips */}
						<div class="mb-3 flex flex-wrap items-center gap-1.5">
							<For each={statuses()}>
								{(s) => {
									const isOn = () => selectedStatuses().has(s.ID);
									const count = () => appsByStatus()[s.ID] ?? 0;
									return (
										<button
											type="button"
											onClick={() => toggleStatus(s.ID)}
											class={cn(
												"flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium transition-colors",
												isOn()
													? ""
													: "border border-border text-faint hover:border-border-strong hover:text-muted",
											)}
											style={
												isOn()
													? {
															"background-color": `color-mix(in srgb, ${s.Colour} 14%, white)`,
															color: `color-mix(in srgb, ${s.Colour} 78%, black)`,
														}
													: {}
											}
										>
											<span
												class="size-2 shrink-0 rounded-full"
												style={
													isOn()
														? { "background-color": s.Colour }
														: {
																border: `1.5px solid ${s.Colour}`,
																"background-color": "transparent",
															}
												}
											/>
											<span>{s.Name}</span>
											<span class="font-mono tabular-nums opacity-70">
												{count()}
											</span>
										</button>
									);
								}}
							</For>
							{/* All / None */}
							<span class="ml-0.5 flex items-center gap-1 text-xs text-faint">
								<button
									type="button"
									class="hover:text-foreground"
									onClick={() =>
										setSelectedStatuses(new Set(statuses().map((s) => s.ID)))
									}
								>
									All
								</button>
								<span>·</span>
								<button
									type="button"
									class="hover:text-foreground"
									onClick={() => setSelectedStatuses(new Set())}
								>
									None
								</button>
							</span>
						</div>

						{/* Bar */}
						<Show
							when={selectedStatuses().size > 0 && chartsReady()}
							fallback={
								<p class="text-sm text-faint">Select at least one stage.</p>
							}
						>
							<div class="relative" style={{ height: "56px" }}>
								<Bar data={pipelineData()} options={stackedBarOptions} />
							</div>
						</Show>
					</Show>
				</div>
			</Card>
		</div>
	);
}
