import { Show } from "solid-js";
import { unknownCount } from "@/lib/scoreRows";
import type { ScoreRow } from "@/types/job";

export function SuitabilityScoreValue(props: {
	score: number | null;
	breakdown?: ScoreRow[] | null | undefined;
	size?: "sm" | "lg";
}) {
	return (
		<Show
			when={props.score != null}
			fallback={<p class="text-xs text-faint">Not yet scored.</p>}
		>
			<div class="flex items-baseline gap-1.5">
				<span
					class={`font-mono font-semibold tabular-nums text-foreground ${props.size === "lg" ? "text-lg" : "text-sm"}`}
				>
					{props.score}
				</span>
				<span class="text-xs text-faint">/ 100</span>
				<Show when={unknownCount(props.breakdown) > 0}>
					<span class="text-2xs text-faint">
						· {unknownCount(props.breakdown)} unknown
					</span>
				</Show>
			</div>
		</Show>
	);
}
