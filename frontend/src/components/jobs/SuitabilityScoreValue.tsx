import { Show } from "solid-js";
import { cn } from "@/lib/utils";
import type { Band } from "@/types/job";
import { BandChip } from "./BandChip";

export function SuitabilityScoreValue(props: {
	score: number | null;
	band?: Band | "" | undefined;
	size?: "sm" | "lg";
}) {
	return (
		<Show
			when={props.score != null}
			fallback={<p class="text-xs text-faint">Not yet scored.</p>}
		>
			<div class="flex items-center gap-2">
				<Show when={props.band || undefined}>
					{(band) => <BandChip band={band()} />}
				</Show>
				<span
					class={cn(
						"font-mono tabular-nums",
						props.band ? "text-xs text-faint" : "text-sm font-semibold",
						!props.band && props.size === "lg" && "text-lg",
					)}
				>
					{props.score}
				</span>
			</div>
		</Show>
	);
}
