import { Show } from "solid-js";
import type { Job } from "@/types/job";

interface Props {
	job: Job;
}

export function JobRowExpander(props: Props) {
	const score = () => props.job.SuitabilityScore;

	return (
		<div class="flex flex-col gap-2.5 px-4 py-3 bg-surface-muted border-t border-border">
			<Show
				when={score() != null}
				fallback={<p class="text-xs text-faint">Not yet scored.</p>}
			>
				<div class="flex items-baseline gap-1">
					<span class="font-mono text-sm font-semibold tabular-nums text-foreground">
						{score()}
					</span>
					<span class="text-xs text-faint">/ 100</span>
				</div>
			</Show>
		</div>
	);
}
