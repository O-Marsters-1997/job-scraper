import { Bar } from "solid-chartjs";
import { createMemo, Show } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	destructiveHex,
	hexAlpha,
	horizontalBarOptions,
	primaryHex,
} from "@/lib/charts";
import type { Job } from "@/types/job";

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

export function SkillGapChart(props: { jobs: Job[]; chartsReady: boolean }) {
	const skillGap = createMemo(() => ({
		missing: topSkills(props.jobs.flatMap((j) => j.Missing ?? [])),
		matched: topSkills(props.jobs.flatMap((j) => j.Matched ?? [])),
	}));

	const missingData = createMemo(() => {
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

	const matchedData = createMemo(() => {
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

	return (
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
						<Show when={props.chartsReady}>
							<div
								class="relative"
								style={{
									height: `${Math.max(120, skillGap().missing.length * 26)}px`,
								}}
							>
								<Bar data={missingData()} options={horizontalBarOptions()} />
							</div>
						</Show>
					</div>
					<div class="px-5 py-4">
						<p class="mb-2 text-xs text-faint">Matched skills</p>
						<Show
							when={skillGap().matched.length > 0 && props.chartsReady}
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
								<Bar data={matchedData()} options={horizontalBarOptions()} />
							</div>
						</Show>
					</div>
				</div>
			</Card>
		</Show>
	);
}
