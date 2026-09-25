import { For, Show } from "solid-js";
import { Badge } from "@/components/ui/badge";
import { useRequestReasoning } from "@/hooks/useJobs";
import type { Job } from "@/types/job";

interface Props {
	job: Job;
}

export function JobRowExpander(props: Props) {
	const reasoningMutation = useRequestReasoning();

	const score = () => props.job.SuitabilityScore;
	const reasoning = () => props.job.Reasoning ?? null;
	const matched = () => props.job.Matched ?? [];
	const missing = () => props.job.Missing ?? [];
	const skipped = () => props.job.SuitabilitySkipped ?? false;

	const isSkipped = () => skipped();
	const isScored = () => !isSkipped() && score() != null && reasoning() != null;
	const isLegacy = () => !isSkipped() && score() != null && reasoning() == null;
	const isPending = () => !isSkipped() && score() == null;

	return (
		<div class="flex flex-col gap-2.5 px-4 py-3 bg-surface-muted border-t border-border">
			<Show when={isSkipped()}>
				<p class="text-xs text-faint">
					Suitability skipped — below relevance cutoff.
				</p>
			</Show>

			<Show when={isScored()}>
				<p class="text-xs leading-relaxed text-muted">{reasoning()}</p>

				<Show when={matched().length > 0}>
					<div class="flex flex-col gap-1">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							Matched
						</span>
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

				<Show when={missing().length > 0}>
					<div class="flex flex-col gap-1">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							Missing
						</span>
						<div class="flex flex-wrap gap-1">
							<For each={missing()}>
								{(item) => <Badge variant="secondary">{item}</Badge>}
							</For>
						</div>
					</div>
				</Show>
			</Show>

			<Show when={isLegacy()}>
				<div class="flex items-center gap-3">
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
				</div>
			</Show>

			<Show when={isPending()}>
				<p class="text-xs text-faint">Not yet scored.</p>
			</Show>
		</div>
	);
}
