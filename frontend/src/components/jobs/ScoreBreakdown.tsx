import { createMemo, For, Show } from "solid-js";
import { MATCHED_COLOUR, MISSING_COLOUR } from "@/lib/scoreColour";
import { unknownCount } from "@/lib/scoreRows";
import type { Job, ScoreRow } from "@/types/job";

function isVisible(row: ScoreRow): boolean {
	if (
		row.effect === "meets" ||
		row.effect === "unknown" ||
		row.effect === "retired"
	)
		return true;
	return row.effect === "misses" && row.stance === "avoid";
}

export function ScoreBreakdown(props: { job: Job }) {
	const rows = createMemo(() => (props.job.Breakdown ?? []).filter(isVisible));
	const matched = createMemo(() => rows().filter((r) => r.effect === "meets"));
	const avoidHits = createMemo(() =>
		rows().filter((r) => r.effect === "misses"),
	);
	const retired = createMemo(() =>
		rows().filter((r) => r.effect === "retired"),
	);
	const unknowns = createMemo(() => unknownCount(props.job.Breakdown));

	return (
		<Show when={rows().length > 0}>
			<div class="flex flex-col gap-2.5">
				<Show when={matched().length > 0}>
					<div class="flex flex-col gap-1">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							Matched
						</span>
						<div class="flex flex-wrap gap-1">
							<For each={matched()}>
								{(r) => (
									<span
										class="inline-flex max-w-[240px] items-center truncate rounded-full px-2.5 py-0.5 text-xs font-medium"
										style={{
											background: `color-mix(in srgb, ${MATCHED_COLOUR} 12%, white)`,
											color: `color-mix(in srgb, ${MATCHED_COLOUR} 80%, black)`,
											border: `1px solid color-mix(in srgb, ${MATCHED_COLOUR} 28%, white)`,
										}}
										title={r.label}
									>
										{r.label}
									</span>
								)}
							</For>
						</div>
					</div>
				</Show>

				<Show when={avoidHits().length > 0}>
					<div class="flex flex-col gap-1">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							Avoid hit
						</span>
						<div class="flex flex-wrap gap-1">
							<For each={avoidHits()}>
								{(r) => (
									<span
										class="inline-flex max-w-[240px] items-center truncate rounded-full px-2.5 py-0.5 text-xs font-medium"
										style={{
											background: `color-mix(in srgb, ${MISSING_COLOUR} 12%, white)`,
											color: `color-mix(in srgb, ${MISSING_COLOUR} 80%, black)`,
											border: `1px solid color-mix(in srgb, ${MISSING_COLOUR} 28%, white)`,
										}}
										title={r.label}
									>
										{r.label}
									</span>
								)}
							</For>
						</div>
					</div>
				</Show>

				<Show when={retired().length > 0}>
					<div class="flex flex-col gap-1">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							Retired
						</span>
						<div class="flex flex-wrap gap-1">
							<For each={retired()}>
								{(r) => (
									<span
										class="inline-flex max-w-[240px] items-center truncate rounded-full bg-surface-muted px-2.5 py-0.5 text-xs font-medium text-muted"
										title={r.label}
									>
										{r.label}
									</span>
								)}
							</For>
						</div>
					</div>
				</Show>

				<Show when={unknowns() > 0}>
					<p class="text-2xs text-faint">
						{unknowns()} unknown — not stated in the posting
					</p>
				</Show>
			</div>
		</Show>
	);
}
