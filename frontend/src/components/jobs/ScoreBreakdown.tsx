import { createSignal, For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { useRevertCorrection, useSetCorrection } from "@/hooks/useCorrections";
import { MATCHED_COLOUR, MISSING_COLOUR, tintedChip } from "@/lib/scoreColour";
import { cn } from "@/lib/utils";
import type { Job, ScoreRow } from "@/types/job";

const CORRECTED_CLASS =
	"bg-accent-subtle text-accent-text border border-accent-border";

const GROUPS: {
	label: string;
	keep: (row: ScoreRow) => boolean;
	style?: ReturnType<typeof tintedChip>;
	class?: string;
}[] = [
	{
		label: "Gates",
		keep: (r) => r.effect === "gated" || r.effect === "blocked",
		style: tintedChip(MISSING_COLOUR),
	},
	{
		label: "Boosts",
		keep: (r) => r.effect === "favourite",
		style: tintedChip(MATCHED_COLOUR),
	},
	{
		label: "Met",
		keep: (r) => r.effect === "meets",
		style: tintedChip(MATCHED_COLOUR),
	},
	{
		label: "Missed",
		keep: (r) => r.effect === "misses",
		style: tintedChip(MISSING_COLOUR),
	},
	{
		label: "Level",
		keep: (r) => r.effect === "level",
		class: "bg-surface text-muted border border-border",
	},
	{
		label: "Unknown",
		keep: (r) => r.effect === "unknown",
		class: "bg-surface-muted text-muted border border-border",
	},
	{
		label: "Corrected",
		keep: (r) => r.corrected === true && r.effect === "neutral",
		class: CORRECTED_CLASS,
	},
	{
		label: "Retired",
		keep: (r) => r.effect === "retired",
		class: "bg-surface-muted text-muted",
	},
];

const canCorrect = (r: ScoreRow) =>
	r.corrected === true ||
	(r.resolved === "yes" &&
		r.effect !== "retired" &&
		r.effect !== "favourite" &&
		r.key.includes(":"));

export function ScoreBreakdown(props: { job: Job }) {
	const [selectedKey, setSelectedKey] = createSignal<string>();
	const setCorrection = useSetCorrection();
	const revertCorrection = useRevertCorrection();
	const breakdown = () => props.job.Breakdown ?? [];
	const selected = () => breakdown().find((r) => r.key === selectedKey());
	const target = (r: ScoreRow) => ({ jobId: props.job.ID, optionId: r.key });
	const groups = () =>
		GROUPS.map((g) => ({ ...g, rows: breakdown().filter(g.keep) })).filter(
			(g) => g.rows.length > 0,
		);

	return (
		<Show when={groups().length > 0}>
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
										<button
											type="button"
											disabled={!canCorrect(r)}
											aria-expanded={selectedKey() === r.key}
											class={cn(
												"inline-flex max-w-[240px] items-center truncate rounded-full px-2.5 py-0.5 text-xs font-medium enabled:cursor-pointer",
												r.corrected ? CORRECTED_CLASS : g.class,
											)}
											style={r.corrected ? undefined : g.style}
											title={r.corrected ? `${r.label} (corrected)` : r.label}
											onClick={() =>
												setSelectedKey(
													selectedKey() === r.key ? undefined : r.key,
												)
											}
										>
											{r.label}
										</button>
									)}
								</For>
							</div>
						</div>
					)}
				</For>
				<Show when={selected()}>
					{(row) => (
						<div class="flex items-center gap-2 text-xs text-muted">
							<Show
								when={row().corrected}
								fallback={
									<Button
										variant="outline"
										size="sm"
										disabled={setCorrection.isPending}
										onClick={() => {
											setCorrection.mutate({
												...target(row()),
												value: row().resolved === "yes" ? "no" : "yes",
											});
											setSelectedKey(undefined);
										}}
									>
										This is wrong
									</Button>
								}
							>
								<span>
									You marked {row().label} as {row().resolved}.
								</span>
								<Button
									variant="ghost"
									size="sm"
									disabled={revertCorrection.isPending}
									onClick={() => {
										revertCorrection.mutate(target(row()));
										setSelectedKey(undefined);
									}}
								>
									Revert
								</Button>
							</Show>
						</div>
					)}
				</Show>
			</div>
		</Show>
	);
}
