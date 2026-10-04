import { createMemo, For, Show } from "solid-js";
import { ladderBand } from "@/lib/ladder";
import type { ScoringOption } from "@/types/scoringOptions";

const TOP_LEVEL = 5;

const percentAt = (level: number) =>
	(Math.min(Math.max(level, 1), TOP_LEVEL) - 1) * (100 / (TOP_LEVEL - 1));

export function SeniorityLadder(props: {
	tiers: ScoringOption[];
	weightOf: (id: string) => number;
	setWeight: (id: string, weight: number) => void;
}) {
	const tiers = () =>
		[...props.tiers].sort((a, b) => (a.level ?? 0) - (b.level ?? 0));
	const band = createMemo(() =>
		ladderBand(
			tiers().map((t) => ({
				level: t.level ?? 0,
				weight: props.weightOf(t.id),
			})),
		),
	);

	return (
		<fieldset>
			<legend class="text-sm font-medium text-foreground">Seniority</legend>
			<p class="mt-0.5 text-xs text-faint">
				Weight each level you'd consider. Postings that don't state a level are
				never penalised.
			</p>
			<div class="mt-2 flex max-w-md flex-col gap-2">
				<For each={tiers()}>
					{(tier) => (
						<label class="flex items-center gap-3 text-sm text-muted">
							<span class="w-32 shrink-0">{tier.label}</span>
							<input
								type="range"
								min="0"
								max="100"
								step="5"
								value={props.weightOf(tier.id)}
								onInput={(e) =>
									props.setWeight(tier.id, Number(e.currentTarget.value))
								}
								class="min-w-0 flex-1 accent-primary"
							/>
							<span class="w-8 text-right font-mono text-xs tabular-nums text-foreground">
								{props.weightOf(tier.id)}
							</span>
						</label>
					)}
				</For>
				<div class="mt-1 flex flex-col gap-1" aria-live="polite">
					<div class="relative h-2 rounded-full bg-surface-muted ring-1 ring-border">
						<Show when={band()}>
							{(b) => (
								<>
									<div
										class="absolute inset-y-0 rounded-full bg-accent-subtle ring-1 ring-accent-border"
										style={{
											left: `${percentAt(b().point - b().tolerance)}%`,
											right: `${100 - percentAt(b().point + b().tolerance)}%`,
										}}
									/>
									<div
										class="absolute -top-1 h-4 w-0.5 -translate-x-1/2 rounded-full bg-primary"
										style={{ left: `${percentAt(b().point)}%` }}
									/>
								</>
							)}
						</Show>
					</div>
					<div class="flex justify-between font-mono text-2xs text-faint tabular-nums">
						<For each={tiers()}>{(tier) => <span>{tier.level}</span>}</For>
					</div>
					<p class="text-xs text-muted">
						<Show
							when={band()}
							fallback="Set a weight to place yourself on the ladder."
						>
							{(b) => (
								<>
									You{" "}
									<span class="font-mono tabular-nums text-foreground">
										{b().point.toFixed(1)} ± {b().tolerance.toFixed(1)}
									</span>
								</>
							)}
						</Show>
					</p>
				</div>
			</div>
		</fieldset>
	);
}
