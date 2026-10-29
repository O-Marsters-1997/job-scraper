import type { CreateQueryResult } from "@tanstack/solid-query";
import { For, Show } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { rowLabel } from "@/lib/scoreRows";
import { cn } from "@/lib/utils";
import { GRADE_LABEL } from "@/types/grade";
import type { ScoreRow } from "@/types/job";
import type { ScoringPreview } from "@/types/scoringPreview";

const REASON_EFFECTS: ScoreRow["effect"][] = [
	"gated",
	"favourite",
	"meets",
	"misses",
];

function reasonTone(effect: ScoreRow["effect"]) {
	switch (effect) {
		case "gated":
			return "font-medium text-destructive-strong";
		case "misses":
			return "text-muted";
		default:
			return "text-accent-text";
	}
}

export function ScorePreview(props: {
	query: CreateQueryResult<ScoringPreview>;
}) {
	return (
		<section class="flex flex-col gap-3" aria-labelledby="preview-heading">
			<div>
				<h2
					id="preview-heading"
					class="text-base font-semibold text-foreground"
				>
					Preview
				</h2>
				<p class="text-xs text-faint">
					Your recent jobs, scored with the settings on this page. Nothing is
					saved.
				</p>
			</div>
			<QueryBoundary query={props.query} fallbackRows={3}>
				{(data) => <PreviewList data={data()} />}
			</QueryBoundary>
		</section>
	);
}

function PreviewList(props: { data: ScoringPreview }) {
	return (
		<Show
			when={props.data.jobs.length > 0}
			fallback={<p class="text-sm text-muted">No jobs to preview yet.</p>}
		>
			<div class="rounded-md border border-border px-3 py-2">
				<p class="text-xs text-faint">Dismissed in top 5</p>
				<p class="font-mono text-sm tabular-nums text-foreground">
					{props.data.dismissedInTop5.before} →{" "}
					{props.data.dismissedInTop5.after}
				</p>
			</div>
			<ol class="flex flex-col rounded-md border border-border">
				<For each={props.data.jobs}>
					{(job) => (
						<li class="flex flex-col gap-1 border-b border-border px-3 py-2.5 last:border-b-0">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									<p class="truncate text-sm font-medium text-foreground">
										{job.title}
									</p>
									<p class="truncate text-xs text-muted">{job.company}</p>
								</div>
								<span class="shrink-0 font-mono text-sm tabular-nums">
									<span class="text-faint line-through">{job.before}</span>{" "}
									<span class="font-medium text-foreground">{job.after}</span>
								</span>
							</div>
							<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
								<Show when={job.grade}>
									{(grade) => (
										<Badge variant="outline">{GRADE_LABEL[grade()]}</Badge>
									)}
								</Show>
								<For
									each={job.afterRows.filter((r) =>
										REASON_EFFECTS.includes(r.effect),
									)}
								>
									{(row) => (
										<span class={cn("text-xs", reasonTone(row.effect))}>
											{rowLabel(row)}
										</span>
									)}
								</For>
							</div>
						</li>
					)}
				</For>
			</ol>
		</Show>
	);
}
