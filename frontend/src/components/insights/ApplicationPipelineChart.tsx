import { Bar } from "solid-chartjs";
import { createMemo, createSignal, For, Show, untrack } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	createThemedMemo,
	hexAlpha,
	stackedBarOptions,
	useChartCanvasRef,
} from "@/lib/charts";
import { cn } from "@/lib/utils";
import type { ApplicationWithDetails } from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";

export function ApplicationPipelineChart(props: {
	applications: ApplicationWithDetails[];
	statuses: ApplicationStatus[];
}) {
	const [selectedStatuses, setSelectedStatuses] = createSignal(
		untrack(() => new Set(props.statuses.map((s) => s.ID))),
	);

	function toggleStatus(id: string) {
		setSelectedStatuses((prev) => {
			const next = new Set(prev);
			if (next.has(id)) next.delete(id);
			else next.add(id);
			return next;
		});
	}

	const appsByStatus = createMemo(() => {
		const map: Record<string, number> = {};
		for (const app of props.applications) {
			map[app.StatusID] = (map[app.StatusID] ?? 0) + 1;
		}
		return map;
	});

	const pipelineData = createMemo(() => ({
		labels: [""],
		datasets: props.statuses
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

	const options = createThemedMemo(stackedBarOptions);

	const totalApps = () => props.applications.length;
	const setCanvas = useChartCanvasRef(
		() =>
			`Stacked bar chart of application pipeline, ${totalApps()} application${totalApps() === 1 ? "" : "s"} across ${selectedStatuses().size} selected stage${selectedStatuses().size === 1 ? "" : "s"}`,
	);

	return (
		<Card>
			<div class="border-b border-border px-5 py-4">
				<h3 class="text-sm font-semibold text-foreground">
					Application pipeline
				</h3>
				<p class="mt-0.5 text-xs text-faint">
					{totalApps()} application{totalApps() === 1 ? "" : "s"} across{" "}
					{props.statuses.length} stage{props.statuses.length === 1 ? "" : "s"}
				</p>
			</div>
			<div class="px-5 py-4">
				<Show
					when={totalApps() > 0}
					fallback={<p class="text-sm text-faint">No applications yet.</p>}
				>
					<div class="mb-3 flex flex-wrap items-center gap-1.5">
						<For each={props.statuses}>
							{(s) => {
								const isOn = () => selectedStatuses().has(s.ID);
								const count = () => appsByStatus()[s.ID] ?? 0;
								return (
									<button
										type="button"
										onClick={() => toggleStatus(s.ID)}
										aria-pressed={isOn()}
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
						<span class="ml-0.5 flex items-center gap-1 text-xs text-faint">
							<button
								type="button"
								class="hover:text-foreground"
								onClick={() =>
									setSelectedStatuses(new Set(props.statuses.map((s) => s.ID)))
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

					<Show
						when={selectedStatuses().size > 0}
						fallback={
							<p class="text-sm text-faint">Select at least one stage.</p>
						}
					>
						<div class="relative" style={{ height: "56px" }}>
							<Bar ref={setCanvas} data={pipelineData()} options={options()} />
						</div>
					</Show>
				</Show>
			</div>
		</Card>
	);
}
