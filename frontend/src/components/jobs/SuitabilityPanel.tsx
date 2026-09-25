import { Show } from "solid-js";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Job } from "@/types/job";

export function SuitabilityPanel(props: { job: Job }) {
	const score = () => props.job.SuitabilityScore;

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Suitability</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<Show
					when={score() != null}
					fallback={<p class="text-xs text-faint">Not yet scored.</p>}
				>
					<div class="flex items-baseline gap-1">
						<span class="font-mono text-lg font-semibold tabular-nums text-foreground">
							{score()}
						</span>
						<span class="text-xs text-faint">/ 100</span>
					</div>
				</Show>
			</CardContent>
		</Card>
	);
}
