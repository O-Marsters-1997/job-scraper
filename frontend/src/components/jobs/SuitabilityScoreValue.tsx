import { Show } from "solid-js";

export function SuitabilityScoreValue(props: {
	score: number | null;
	size?: "sm" | "lg";
}) {
	return (
		<Show
			when={props.score != null}
			fallback={<p class="text-xs text-faint">Not yet scored.</p>}
		>
			<div class="flex items-baseline gap-1">
				<span
					class={`font-mono font-semibold tabular-nums text-foreground ${props.size === "lg" ? "text-lg" : "text-sm"}`}
				>
					{props.score}
				</span>
				<span class="text-xs text-faint">/ 100</span>
			</div>
		</Show>
	);
}
