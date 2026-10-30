import { createFileRoute, Link } from "@tanstack/solid-router";
import { createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { createStore } from "solid-js/store";
import { BandChip } from "@/components/jobs/BandChip";
import JobDescription from "@/components/jobs/JobDescription";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { rowLabel } from "@/lib/scoreRows";
import type { Band, Job } from "@/types/job";
import { useJob } from "../../hooks/useJobs";
import {
	usePlaygroundQueue,
	usePlaygroundVerdict,
} from "../../hooks/useScoringPlayground";
import type { JobFeedbackInput } from "../../types/scoreFeedback";
import type { PlaygroundQueue } from "../../types/scoringPlayground";

import { isTyping } from "./-playground/isTyping";
import { buildReason } from "./-playground/reason";
import { Signals } from "./-playground/Signals";
import { type HistoryEntry, pendingIds } from "./-playground/session";
import { Verdict } from "./-playground/Verdict";

export const Route = createFileRoute("/_auth/scoring-playground")({
	component: ScoringPlayground,
});

function ScoringPlayground() {
	const queue = usePlaygroundQueue();
	return (
		<div class="flex min-h-dvh flex-col gap-4 px-7 py-6 pb-20">
			<QueryBoundary query={queue} fallbackRows={4}>
				{(data) => <Session queue={data()} />}
			</QueryBoundary>
		</div>
	);
}

function Session(props: { queue: PlaygroundQueue }) {
	const verdict = usePlaygroundVerdict();
	const [history, setHistory] = createSignal<HistoryEntry[]>([]);
	const [skipped, setSkipped] = createSignal<string[]>([]);
	const [pos, setPos] = createSignal<number>();
	const pending = () => pendingIds(props.queue.jobIds, skipped());
	const entry = () => {
		const p = pos();
		return p === undefined ? undefined : history()[p];
	};
	const jobId = () => entry()?.jobId ?? pending()[0];
	const total = () => history().length + pending().length;
	const canBack = () => !verdict.isPending && (pos() ?? history().length) > 0;
	const canForward = () =>
		!verdict.isPending &&
		(pos() !== undefined || (pending().length > 1 && !!jobId()));
	const back = () => {
		if (canBack()) setPos((pos() ?? history().length) - 1);
	};
	const forward = () => {
		if (!canForward()) return;
		const p = pos();
		if (p === undefined) {
			const id = jobId();
			if (id) setSkipped([...skipped().filter((s) => s !== id), id]);
		} else setPos(p + 1 < history().length ? p + 1 : undefined);
	};
	const onKey = (e: KeyboardEvent) => {
		if (e.metaKey || e.ctrlKey || e.altKey || isTyping(e.target)) return;
		if (e.key === "ArrowLeft") back();
		else if (e.key === "ArrowRight") forward();
	};
	onMount(() => window.addEventListener("keydown", onKey));
	onCleanup(() => window.removeEventListener("keydown", onKey));
	const save = (
		id: string,
		input: JobFeedbackInput,
		draft: HistoryEntry["draft"],
	) => {
		const p = pos();
		verdict.mutate(
			{ ...input, replaces: entry()?.feedbackId },
			{
				onSuccess: (saved) => {
					const next = {
						jobId: id,
						feedbackId: saved.id,
						verdict: input.direction,
						draft,
					};
					if (p === undefined) setHistory([...history(), next]);
					else {
						setHistory(history().map((h, i) => (i === p ? next : h)));
						setPos(p + 1 < history().length ? p + 1 : undefined);
					}
				},
			},
		);
	};
	return (
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
				<nav
					class="flex items-center gap-2 text-sm text-muted"
					aria-label="Job history"
				>
					<Button
						variant="ghost"
						size="sm"
						aria-label="Previous job"
						disabled={!canBack()}
						onClick={back}
					>
						‹
					</Button>
					<span class="font-mono tabular-nums">
						Job {(pos() ?? history().length) + 1} of {Math.max(total(), 1)}
					</span>
					<Button
						variant="ghost"
						size="sm"
						aria-label="Next job"
						disabled={!canForward()}
						onClick={forward}
					>
						›
					</Button>
				</nav>
				<Progress bands={props.queue.bands} />
			</header>
			<Show when={jobId()} fallback={<CaughtUp />} keyed>
				{(id) => (
					<Review
						jobId={id}
						entry={entry()}
						failed={verdict.isError && verdict.variables?.jobId === id}
						onSave={(input, draft) => save(id, input, draft)}
					/>
				)}
			</Show>
		</>
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

function Review(props: {
	jobId: string;
	entry: HistoryEntry | undefined;
	failed: boolean;
	onSave: (input: JobFeedbackInput, draft: HistoryEntry["draft"]) => void;
}) {
	const job = useJob(() => props.jobId);
	const [notes, setNotes] = createStore<Record<string, string>>({
		...props.entry?.draft.notes,
	});
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
							jobId={data().ID}
							rows={data().Breakdown ?? []}
							notes={notes}
							onNote={(key, note) => setNotes(key, note)}
						/>
						<Verdict
							failed={props.failed}
							current={props.entry?.verdict}
							initialOverall={props.entry?.draft.overall}
							onVerdict={(direction, overall) =>
								props.onSave(
									{
										jobId: props.jobId,
										direction,
										reason: buildReason(
											overall,
											(data().Breakdown ?? []).map((r) => ({
												label: rowLabel(r),
												note: notes[r.key] ?? "",
											})),
										),
									},
									{ overall, notes: { ...notes } },
								)
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
