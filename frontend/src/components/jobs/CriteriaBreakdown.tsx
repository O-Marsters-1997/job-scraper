import { createMemo, For, Show } from "solid-js";
import { Badge } from "@/components/ui/badge";
import { useScoringConfig } from "@/hooks/useScoringConfig";
import { criterionScores } from "@/lib/criteria";
import { MATCHED_COLOUR, MISSING_COLOUR } from "@/lib/scoreColour";
import type { Job } from "@/types/job";

export function CriteriaBreakdown(props: { job: Job }) {
	const scoringConfig = useScoringConfig();

	const labelFor = (key: string): string =>
		scoringConfig.data?.scoringQuestions.criteria.find((c) => c.key === key)
			?.instructions ?? key;

	const scores = createMemo(() =>
		criterionScores(props.job.Criteria).map((c) => ({
			...c,
			label: labelFor(c.key),
			pct: Math.round(c.probability * 100),
		})),
	);
	const matched = createMemo(() => scores().filter((c) => c.matched));
	const missing = createMemo(() => scores().filter((c) => !c.matched));

	return (
		<Show when={scores().length > 0}>
			<div class="flex flex-col gap-2.5">
				<For each={scores()}>
					{(c) => (
						<div class="flex flex-col gap-1">
							<div class="flex items-baseline justify-between gap-2">
								<span class="truncate text-xs text-muted" title={c.label}>
									{c.label}
								</span>
								<span class="shrink-0 font-mono text-xs tabular-nums text-faint">
									{c.pct}%
								</span>
							</div>
							<div
								class="h-1.5 w-full overflow-hidden rounded-full bg-surface-muted"
								role="progressbar"
								aria-label={c.label}
								aria-valuenow={c.pct}
								aria-valuemin={0}
								aria-valuemax={100}
							>
								<div
									class="h-full rounded-full"
									style={{
										width: `${c.pct}%`,
										"background-color": c.matched
											? MATCHED_COLOUR
											: MISSING_COLOUR,
									}}
								/>
							</div>
						</div>
					)}
				</For>

				<Show when={matched().length > 0}>
					<div class="flex flex-col gap-1">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							Matched
						</span>
						<div class="flex flex-wrap gap-1">
							<For each={matched()}>
								{(c) => (
									<span
										class="inline-flex max-w-[240px] items-center truncate rounded-full px-2.5 py-0.5 text-xs font-medium"
										style={{
											background: `color-mix(in srgb, ${MATCHED_COLOUR} 12%, white)`,
											color: `color-mix(in srgb, ${MATCHED_COLOUR} 80%, black)`,
											border: `1px solid color-mix(in srgb, ${MATCHED_COLOUR} 28%, white)`,
										}}
										title={c.label}
									>
										{c.label}
									</span>
								)}
							</For>
						</div>
					</div>
				</Show>

				<Show when={missing().length > 0}>
					<div class="flex flex-col gap-1">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							Missing
						</span>
						<div class="flex flex-wrap gap-1">
							<For each={missing()}>
								{(c) => (
									<Badge
										variant="secondary"
										class="max-w-[240px] truncate"
										title={c.label}
									>
										{c.label}
									</Badge>
								)}
							</For>
						</div>
					</div>
				</Show>
			</div>
		</Show>
	);
}
