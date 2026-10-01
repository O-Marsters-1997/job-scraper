import { For, Show } from "solid-js";
import { MATCHED_COLOUR, MISSING_COLOUR } from "@/lib/scoreColour";
import { unknownCount } from "@/lib/scoreRows";
import { cn } from "@/lib/utils";
import type { Job, ScoreRow } from "@/types/job";

const tintedChip = (colour: string) => ({
	background: `color-mix(in srgb, ${colour} 12%, white)`,
	color: `color-mix(in srgb, ${colour} 80%, black)`,
	border: `1px solid color-mix(in srgb, ${colour} 28%, white)`,
});

const GROUPS: {
	label: string;
	keep: (row: ScoreRow) => boolean;
	style?: ReturnType<typeof tintedChip>;
	class?: string;
}[] = [
	{
		label: "Blocked",
		keep: (r) => r.effect === "blocked",
		style: tintedChip(MISSING_COLOUR),
	},
	{
		label: "Matched",
		keep: (r) => r.effect === "meets",
		style: tintedChip(MATCHED_COLOUR),
	},
	{
		label: "Avoid hit",
		keep: (r) => r.effect === "misses" && r.stance === "avoid",
		style: tintedChip(MISSING_COLOUR),
	},
	{
		label: "Retired",
		keep: (r) => r.effect === "retired",
		class: "bg-surface-muted text-muted",
	},
];

export function ScoreBreakdown(props: { job: Job }) {
	const breakdown = () => props.job.Breakdown ?? [];
	const groups = () =>
		GROUPS.map((g) => ({ ...g, rows: breakdown().filter(g.keep) })).filter(
			(g) => g.rows.length > 0,
		);
	const unknowns = () => unknownCount(props.job.Breakdown);

	return (
		<Show when={groups().length > 0 || unknowns() > 0}>
			<div class="flex flex-col gap-2.5">
				<For each={groups()}>
					{(g) => (
						<div class="flex flex-col gap-1">
							<span class="text-2xs font-medium uppercase tracking-wide text-faint">
								{g.label}
							</span>
							<div class="flex flex-wrap gap-1">
								<For each={g.rows}>
									{(r) => (
										<span
											class={cn(
												"inline-flex max-w-[240px] items-center truncate rounded-full px-2.5 py-0.5 text-xs font-medium",
												g.class,
											)}
											style={g.style}
											title={r.label}
										>
											{r.label}
										</span>
									)}
								</For>
							</div>
						</div>
					)}
				</For>
				<Show when={unknowns() > 0}>
					<p class="text-2xs text-faint">
						{unknowns()} unknown — not stated in the posting
					</p>
				</Show>
			</div>
		</Show>
	);
}
