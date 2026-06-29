import { createFileRoute } from "@tanstack/solid-router";
import { Chart as ChartJS } from "chart.js";
import { Bar, Doughnut, Line } from "solid-chartjs";
import {
	createEffect,
	createMemo,
	createSignal,
	For,
	onMount,
	Show,
} from "solid-js";
import { Card } from "@/components/ui/card";
import {
	destructiveHex,
	donutChartOptions,
	hexAlpha,
	horizontalBarOptions,
	lineChartOptions,
	primaryHex,
	registerCharts,
	sourceHex,
	stackedBarOptions,
	verticalBarOptions,
} from "@/lib/charts";
import { cn } from "@/lib/utils";
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

const SCORE_BANDS = ["0–19", "20–39", "40–59", "60–79", "80–100"];

function topSkills(arr: string[]): [string, number][] {
	const counts: Record<string, number> = {};
	for (const s of arr) {
		const key = s.trim().toLowerCase();
		if (key) counts[key] = (counts[key] ?? 0) + 1;
	}
	return Object.entries(counts)
		.sort((a, b) => b[1] - a[1])
		.slice(0, 10);
}

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

	// ── A4: Score distribution + gate stats ───────────────────────────────────
	const gateStats = createMemo(() => {
		let scored = 0;
		let gated = 0;
		let unscored = 0;
		for (const job of jobs()) {
			if (job.SuitabilityScore != null) scored++;
			else if (job.SuitabilitySkipped) gated++;
			else unscored++;
		}
		return { scored, gated, unscored };
	});

	const hasScores = () => gateStats().scored > 0;

	const scoreDistData = createMemo(() => {
		const bands = [0, 0, 0, 0, 0];
		for (const job of jobs()) {
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

	// ── A5: Skill gap ─────────────────────────────────────────────────────────
	const skillGap = createMemo(() => ({
		missing: topSkills(jobs().flatMap((j) => j.Missing ?? [])),
		matched: topSkills(jobs().flatMap((j) => j.Matched ?? [])),
	}));

	const missingSkillsData = createMemo(() => {
		const entries = skillGap().missing;
		const color = destructiveHex();
		return {
			labels: entries.map(([k]) => k),
			datasets: [
				{
					data: entries.map(([, n]) => n),
					backgroundColor: hexAlpha(color, "26"),
					borderColor: color,
					borderWidth: 1.5,
					borderRadius: 3,
				},
			],
		};
	});

	const matchedSkillsData = createMemo(() => {
		const entries = skillGap().matched;
		const color = primaryHex();
		return {
			labels: entries.map(([k]) => k),
			datasets: [
				{
					data: entries.map(([, n]) => n),
					backgroundColor: hexAlpha(color, "26"),
					borderColor: color,
					borderWidth: 1.5,
					borderRadius: 3,
				},
			],
		};
	});

	// ── A6: Source quality ────────────────────────────────────────────────────
	const sourceQualityData = createMemo(() => {
		const scoreSum: Record<string, number> = {};
		const scoreCount: Record<string, number> = {};
		for (const job of jobs()) {
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
								options={lineChartOptions()}
							/>
						</Show>
					</div>
					<p class="px-5 pb-3 pt-1.5 text-2xs text-faint">
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
							<Doughnut
								data={jobsBySourceData()}
								options={donutChartOptions()}
							/>
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
								<Bar data={pipelineData()} options={stackedBarOptions()} />
							</div>
						</Show>
					</Show>
				</div>
			</Card>

			{/* A4 — Score distribution + gate stats */}
			<Card class="mb-3 mt-3">
				<div class="border-b border-border px-5 py-4">
					<h3 class="text-sm font-semibold text-foreground">
						Score &amp; gate
					</h3>
					<p class="mt-0.5 text-xs text-faint">
						How the relevance gate and suitability scorer shaped your feed
					</p>
				</div>
				<div class="grid grid-cols-1 lg:grid-cols-[1fr_180px]">
					<div class="px-5 py-4 lg:border-r lg:border-border">
						<p class="mb-2 text-xs text-faint">
							Suitability score distribution
						</p>
						<Show
							when={hasScores() && chartsReady()}
							fallback={<p class="text-sm text-faint">No scored jobs yet.</p>}
						>
							<div class="relative" style={{ height: "140px" }}>
								<Bar data={scoreDistData()} options={verticalBarOptions()} />
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
								<p class="text-xs font-medium text-muted">Gated</p>
								<p class="mt-0.5 text-xs text-faint">
									Blocked by relevance cutoff
								</p>
							</div>
							<p class="font-mono text-lg font-medium tabular-nums text-foreground">
								{gateStats().gated}
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

			{/* A5 — Source quality */}
			<Card class="mb-3">
				<div class="border-b border-border px-5 py-4">
					<h3 class="text-sm font-semibold text-foreground">Source quality</h3>
					<p class="mt-0.5 text-xs text-faint">
						Average suitability score per job board
					</p>
				</div>
				<div class="px-5 py-4">
					<Show
						when={
							hasScores() &&
							sourceQualityData().labels.length > 0 &&
							chartsReady()
						}
						fallback={<p class="text-sm text-faint">No scored jobs yet.</p>}
					>
						<div
							class="relative"
							style={{
								height: `${Math.max(80, sourceQualityData().labels.length * 40)}px`,
							}}
						>
							<Bar
								data={sourceQualityData()}
								options={horizontalBarOptions()}
							/>
						</div>
					</Show>
				</div>
			</Card>

			{/* A6 — Skill gap (only shown when missing skill data exists) */}
			<Show when={skillGap().missing.length > 0}>
				<Card>
					<div class="border-b border-border px-5 py-4">
						<h3 class="text-sm font-semibold text-foreground">Skill gap</h3>
						<p class="mt-0.5 text-xs text-faint">
							Most common gaps and strengths across scored jobs
						</p>
					</div>
					<div class="grid grid-cols-1 lg:grid-cols-2">
						<div class="px-5 py-4 lg:border-r lg:border-border">
							<p class="mb-2 text-xs text-faint">Missing skills</p>
							<Show when={chartsReady()}>
								<div
									class="relative"
									style={{
										height: `${Math.max(120, skillGap().missing.length * 26)}px`,
									}}
								>
									<Bar
										data={missingSkillsData()}
										options={horizontalBarOptions()}
									/>
								</div>
							</Show>
						</div>
						<div class="px-5 py-4">
							<p class="mb-2 text-xs text-faint">Matched skills</p>
							<Show
								when={skillGap().matched.length > 0 && chartsReady()}
								fallback={
									<p class="text-sm text-faint">No matched skill data.</p>
								}
							>
								<div
									class="relative"
									style={{
										height: `${Math.max(120, skillGap().matched.length * 26)}px`,
									}}
								>
									<Bar
										data={matchedSkillsData()}
										options={horizontalBarOptions()}
									/>
								</div>
							</Show>
						</div>
					</div>
				</Card>
			</Show>
		</div>
	);
}
