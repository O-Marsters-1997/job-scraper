import { Show } from "solid-js";
import { isLowConfidence } from "@/lib/criteria";
import { WARNING_COLOUR } from "@/lib/scoreColour";

export function SuitabilityScoreValue(props: {
	score: number | null;
	confidence?: number | null | undefined;
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
				<Show when={isLowConfidence(props.confidence)}>
					<span
						class="inline-flex items-center rounded-full px-2 py-0.5 text-2xs font-medium"
						style={{
							background: `color-mix(in srgb, ${WARNING_COLOUR} 12%, white)`,
							color: `color-mix(in srgb, ${WARNING_COLOUR} 80%, black)`,
						}}
					>
						Low confidence
					</span>
				</Show>
			</div>
		</Show>
	);
}
