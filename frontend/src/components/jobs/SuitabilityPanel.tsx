import { For, Show } from "solid-js";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useRequestReasoning } from "@/hooks/useJobs";
import type { Job } from "@/types/job";

export function SuitabilityPanel(props: { job: Job }) {
	const reasoningMutation = useRequestReasoning();

	const score = () => props.job.SuitabilityScore;
	const reasoning = () => props.job.Reasoning ?? null;
	const matched = () => props.job.Matched ?? [];
	const missing = () => props.job.Missing ?? [];
	const skipped = () => props.job.SuitabilitySkipped ?? false;

	// skipped — below relevance cutoff, no Claude call made
	const isSkipped = () => skipped() && score() == null;
	// scored — has score + reasoning
	const isScored = () => !isSkipped() && score() != null && reasoning() != null;
	// scored, no reasoning yet — show explain button
	const isLegacy = () => !isSkipped() && score() != null && reasoning() == null;
	// pending — no score, not skipped
	const isPending = () => !isSkipped() && score() == null;

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Suitability</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<Show when={isSkipped()}>
					<p class="rounded-md border border-border bg-surface-muted px-3 py-2 text-xs leading-relaxed text-muted">
						Suitability scoring skipped — this job's relevance score was below
						your cutoff.
					</p>
				</Show>

				<Show when={isScored()}>
					{/* Score display */}
					<div class="flex items-baseline gap-1">
						<span class="font-mono text-lg font-semibold tabular-nums text-foreground">
							{score()}
						</span>
						<span class="text-xs text-faint">/ 100</span>
					</div>

					{/* Rationale */}
					<p class="rounded-md bg-surface-muted px-3 py-2 text-xs leading-relaxed text-muted">
						{reasoning()}
					</p>

					{/* Matched chips */}
					<Show when={matched().length > 0}>
						<div class="flex flex-col gap-1.5">
							<span class="text-xs font-medium text-faint">Matched</span>
							<div class="flex flex-wrap gap-1">
								<For each={matched()}>
									{(item) => (
										<span
											class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
											style={{
												background: "color-mix(in srgb, #059669 12%, white)",
												color: "color-mix(in srgb, #059669 80%, black)",
												border:
													"1px solid color-mix(in srgb, #059669 28%, white)",
											}}
										>
											{item}
										</span>
									)}
								</For>
							</div>
						</div>
					</Show>

					{/* Missing chips */}
					<Show when={missing().length > 0}>
						<div class="flex flex-col gap-1.5">
							<span class="text-xs font-medium text-faint">Missing</span>
							<div class="flex flex-wrap gap-1">
								<For each={missing()}>
									{(item) => <Badge variant="secondary">{item}</Badge>}
								</For>
							</div>
						</div>
					</Show>
				</Show>

				<Show when={isLegacy()}>
					<div class="flex items-baseline gap-1">
						<span class="font-mono text-lg font-semibold tabular-nums text-foreground">
							{score()}
						</span>
						<span class="text-xs text-faint">/ 100</span>
					</div>
					<Show
						when={!reasoningMutation.isError}
						fallback={
							<p class="text-xs text-destructive-strong">
								Failed to generate reasoning. Please try again.
							</p>
						}
					>
						<button
							type="button"
							onClick={() => reasoningMutation.mutate(props.job.ID)}
							disabled={reasoningMutation.isPending}
							class="inline-flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-xs font-medium text-muted transition hover:border-border-strong hover:text-foreground disabled:opacity-50"
						>
							{reasoningMutation.isPending ? "Generating…" : "Explain score"}
						</button>
					</Show>
				</Show>

				<Show when={isPending()}>
					<p class="text-xs text-faint">Not yet scored.</p>
				</Show>
			</CardContent>
		</Card>
	);
}
