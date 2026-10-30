import { createFileRoute, Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { createStore } from "solid-js/store";
import { BandChip } from "@/components/jobs/BandChip";
import JobDescription from "@/components/jobs/JobDescription";
import { QueryBoundary } from "@/components/QueryBoundary";
import { rowLabel } from "@/lib/scoreRows";
import type { Band, Job } from "@/types/job";
import { useJob } from "../../hooks/useJobs";
import {
	usePlaygroundQueue,
	usePlaygroundVerdict,
} from "../../hooks/useScoringPlayground";
import type { PlaygroundQueue } from "../../types/scoringPlayground";
import { buildReason } from "./-playground/reason";
import { Signals } from "./-playground/Signals";
import { Verdict } from "./-playground/Verdict";

export const Route = createFileRoute("/_auth/scoring-playground")({
	component: ScoringPlayground,
});

function ScoringPlayground() {
	const queue = usePlaygroundQueue();
	return (
		<div class="flex min-h-dvh flex-col gap-4 px-7 py-6 pb-20">
			<QueryBoundary query={queue} fallbackRows={4}>
				{(data) => (
					<>
						<header class="flex items-center justify-between gap-4">
							<div class="flex items-center gap-3">
								<Link
									to="/settings/scoring/$section"
									params={{ section: "role" }}
									class="text-sm text-muted hover:text-foreground"
								>
									← Scoring
								</Link>
								<h1 class="text-lg font-bold tracking-tight text-foreground">
									Scoring playground
								</h1>
							</div>
							<Progress bands={data().bands} />
						</header>
						<Show when={data().jobIds[0]} fallback={<CaughtUp />} keyed>
							{(jobId) => <Review jobId={jobId} />}
						</Show>
					</>
				)}
			</QueryBoundary>
		</div>
	);
}

function Progress(props: { bands: PlaygroundQueue["bands"] }) {
	return (
		<div class="flex items-center gap-3 text-xs text-muted">
			<For each={props.bands}>
				{(b) => (
					<span class="flex items-center gap-1">
						<BandChip band={b.band} />
						<span class="font-mono tabular-nums">
							{b.reviewed}/{b.reviewed + b.remaining}
						</span>
					</span>
				)}
			</For>
		</div>
	);
}

function CaughtUp() {
	return (
		<div class="flex flex-col items-center gap-2 py-24 text-center">
			<p class="text-base font-semibold text-foreground">All caught up</p>
			<p class="text-sm text-muted">
				Every scored job from the last 30 days has feedback.
			</p>
		</div>
	);
}

function Review(props: { jobId: string }) {
	const job = useJob(() => props.jobId);
	const verdict = usePlaygroundVerdict();
	const [notes, setNotes] = createStore<Record<string, string>>({});
	return (
		<Show
			when={job.data}
			fallback={<p class="text-sm text-muted">Loading job…</p>}
		>
			{(data) => (
				<div class="grid grid-cols-[minmax(0,3fr)_minmax(0,2fr)] items-start gap-6">
					<article class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-6">
						<JobHeader job={data()} />
						<Show
							when={data().Description}
							fallback={<p class="text-sm text-faint">No description.</p>}
						>
							{(html) => <JobDescription html={html()} />}
						</Show>
					</article>
					<aside class="sticky top-6 flex max-h-[calc(100dvh-3rem)] flex-col gap-4 overflow-y-auto rounded-xl border border-border bg-surface p-4">
						<div class="flex items-center gap-2">
							<span class="font-mono text-2xl font-semibold tabular-nums text-foreground">
								{data().SuitabilityScore}
							</span>
							<Show when={data().Band}>
								{(band) => <BandChip band={band() as Band} />}
							</Show>
						</div>
						<Signals
							rows={data().Breakdown ?? []}
							notes={notes}
							onNote={(key, note) => setNotes(key, note)}
						/>
						<Verdict
							failed={verdict.isError}
							onVerdict={(direction, overall) =>
								verdict.mutate({
									jobId: props.jobId,
									direction,
									reason: buildReason(
										overall,
										(data().Breakdown ?? []).map((r) => ({
											label: rowLabel(r),
											note: notes[r.key] ?? "",
										})),
									),
								})
							}
						/>
					</aside>
				</div>
			)}
		</Show>
	);
}

function JobHeader(props: { job: Job }) {
	return (
		<div class="flex flex-col gap-1">
			<h2 class="text-base font-semibold text-foreground">{props.job.Title}</h2>
			<p class="text-sm text-muted">
				{[
					props.job.CompanySlug,
					props.job.Location,
					props.job.WorkArrangement,
					props.job.SalaryRaw,
				]
					.filter(Boolean)
					.join(" · ")}
			</p>
		</div>
	);
}
