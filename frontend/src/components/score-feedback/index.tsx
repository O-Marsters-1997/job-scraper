import { useQueryClient } from "@tanstack/solid-query";
import {
	createEffect,
	createSignal,
	For,
	type JSX,
	on,
	onCleanup,
	Show,
} from "solid-js";
import { keys } from "@/api/keys";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import {
	useAppendCollectionFeedback,
	useAppendJobFeedback,
	useAppendOverallFeedback,
	useDeleteScoreFeedback,
	useScoreFeedback,
} from "@/hooks/useScoreFeedback";
import type { Job, ScoreRow } from "@/types/job";
import {
	type FeedbackDirection,
	SCORE_FEEDBACK_PAGE_SIZE,
	type ScoreFeedback,
	type ScoreFeedbackKind,
} from "@/types/scoreFeedback";

const KIND_CHIPS: { label: string; kind?: ScoreFeedbackKind }[] = [
	{ label: "All" },
	{ label: "Jobs", kind: "job" },
	{ label: "Collections", kind: "collection" },
	{ label: "Overall", kind: "overall" },
];

const TARGET_SELECTOR = "[data-feedback-job], [data-feedback-collection]";

type JobTarget = {
	id: string;
	title: string;
	score: number;
	breakdown: ScoreRow[];
};

type CollectionTarget = { jobIds: string[]; filters: Record<string, string> };

type Composing =
	| { kind: "overall" }
	| { kind: "job"; target: JobTarget }
	| { kind: "collection"; target: CollectionTarget };

const DIRECTIONS: { value: FeedbackDirection; label: string }[] = [
	{ value: "higher", label: "Should be higher" },
	{ value: "lower", label: "Should be lower" },
];

function targetAt(event: Event): HTMLElement | null {
	return event.target instanceof Element
		? event.target.closest<HTMLElement>(TARGET_SELECTOR)
		: null;
}

export default function ScoreFeedbackPanel() {
	const queryClient = useQueryClient();
	const [open, setOpen] = createSignal(false);
	const [targeting, setTargeting] = createSignal(false);
	const [hovered, setHovered] = createSignal<DOMRect | null>(null);
	const [composing, setComposing] = createSignal<Composing | null>(null);
	const [reason, setReason] = createSignal("");
	const [direction, setDirection] = createSignal<FeedbackDirection | null>(
		null,
	);
	const [kind, setKind] = createSignal<ScoreFeedbackKind | undefined>();
	const [page, setPage] = createSignal(1);
	const [showOutdated, setShowOutdated] = createSignal(false);
	const feedback = useScoreFeedback(kind, page, showOutdated);
	const pill = useScoreFeedback(
		() => undefined,
		() => 1,
	);
	const appendOverall = useAppendOverallFeedback();
	const appendJob = useAppendJobFeedback();
	const appendCollection = useAppendCollectionFeedback();
	const lastPage = () =>
		Math.max(
			1,
			Math.ceil((feedback.data?.total ?? 0) / SCORE_FEEDBACK_PAGE_SIZE),
		);

	createEffect(() => {
		if (feedback.data && page() > lastPage()) setPage(lastPage());
	});

	const pickKind = (next: ScoreFeedbackKind | undefined) => {
		setKind(next);
		setPage(1);
	};

	function cachedJob(id: string): Job | undefined {
		return (
			queryClient.getQueryData<Job>(keys.jobs.detail(id)) ??
			queryClient
				.getQueryData<Job[]>(keys.jobs.full())
				?.find((j) => j.ID === id)
		);
	}

	function startJob(id: string) {
		const job = cachedJob(id);
		if (job?.SuitabilityScore == null) return;
		setComposing({
			kind: "job",
			target: {
				id,
				title: job.Title,
				score: job.SuitabilityScore,
				breakdown: job.Breakdown ?? [],
			},
		});
	}

	function startCollection(ids: string) {
		setComposing({
			kind: "collection",
			target: {
				jobIds: ids === "" ? [] : ids.split(","),
				filters: Object.fromEntries(new URLSearchParams(location.search)),
			},
		});
	}

	function close() {
		setComposing(null);
		setReason("");
		setDirection(null);
	}

	const form = useFormSubmit(async () => {
		const current = composing();
		if (current?.kind === "overall") {
			await appendOverall.mutateAsync({ reason: reason() });
		} else if (current?.kind === "collection") {
			await appendCollection.mutateAsync({
				...current.target,
				reason: reason(),
			});
		} else if (current?.kind === "job") {
			const dir = direction();
			if (dir == null) return;
			await appendJob.mutateAsync({
				jobId: current.target.id,
				direction: dir,
				reason: reason(),
			});
		}
		close();
	});

	createEffect(
		on(targeting, (active) => {
			if (!active) {
				setHovered(null);
				return;
			}
			const onMove = (e: MouseEvent) =>
				setHovered(targetAt(e)?.getBoundingClientRect() ?? null);
			const onClick = (e: MouseEvent) => {
				const el = targetAt(e);
				if (!el) return;
				e.preventDefault();
				e.stopPropagation();
				setTargeting(false);
				if (el.dataset.feedbackJob !== undefined) {
					startJob(el.dataset.feedbackJob);
				} else {
					startCollection(el.dataset.feedbackCollection ?? "");
				}
			};
			const onKey = (e: KeyboardEvent) => {
				if (e.key === "Escape") setTargeting(false);
			};
			document.addEventListener("mousemove", onMove);
			document.addEventListener("click", onClick, true);
			document.addEventListener("keydown", onKey);
			onCleanup(() => {
				document.removeEventListener("mousemove", onMove);
				document.removeEventListener("click", onClick, true);
				document.removeEventListener("keydown", onKey);
			});
		}),
	);

	const canSubmit = () =>
		!form.pending() &&
		reason().trim() !== "" &&
		(composing()?.kind !== "job" || direction() != null);

	return (
		<>
			<button
				type="button"
				class="fixed right-4 bottom-16 z-40 cursor-pointer rounded-full border border-border bg-surface px-3 py-1.5 text-xs font-medium text-muted shadow-sm hover:border-border-strong hover:text-foreground"
				onClick={() => setOpen(true)}
			>
				Score feedback
				<Show when={pill.data}>{(p) => ` (${p().currentCount})`}</Show>
			</button>
			<Show when={hovered()}>
				{(rect) => (
					<div
						class="pointer-events-none fixed z-40 rounded-sm outline-2 outline-primary"
						style={{
							top: `${rect().top}px`,
							left: `${rect().left}px`,
							width: `${rect().width}px`,
							height: `${rect().height}px`,
						}}
					/>
				)}
			</Show>
			<Show when={open()}>
				<aside
					aria-label="Score feedback"
					class="fixed inset-y-0 right-0 z-50 flex w-96 max-w-full flex-col gap-4 overflow-y-auto border-l border-border bg-surface p-5"
				>
					<header class="flex items-center justify-between">
						<h2 class="text-base font-semibold">Score feedback</h2>
						<Button variant="ghost" size="sm" onClick={() => setOpen(false)}>
							Close
						</Button>
					</header>

					<Show
						when={composing()}
						fallback={
							<div class="flex gap-2">
								<Button
									variant={targeting() ? "default" : "outline"}
									onClick={() => setTargeting(!targeting())}
								>
									{targeting()
										? "Pick a Job or list (Esc to cancel)"
										: "Target"}
								</Button>
								<Button
									variant="outline"
									onClick={() => setComposing({ kind: "overall" })}
								>
									Overall
								</Button>
							</div>
						}
					>
						{(current) => (
							<form class="flex flex-col gap-3" onSubmit={form.submit}>
								<FormFeedback error={form.error()} />
								<Show when={jobTarget(current())}>
									{(target) => <JobEvidence target={target()} />}
								</Show>
								<Show when={collectionTarget(current())}>
									{(target) => <CollectionSummary target={target()} />}
								</Show>
								<Show when={current().kind === "job"}>
									<div class="flex gap-2" role="radiogroup">
										<For each={DIRECTIONS}>
											{(d) => (
												<Button
													type="button"
													variant={
														direction() === d.value ? "default" : "outline"
													}
													aria-pressed={direction() === d.value}
													onClick={() => setDirection(d.value)}
												>
													{d.label}
												</Button>
											)}
										</For>
									</div>
								</Show>
								<div class="flex flex-col gap-1 text-xs text-faint">
									<label for="score-feedback-reason">Reason</label>
									<Textarea
										id="score-feedback-reason"
										class="min-h-24 text-sm text-foreground"
										value={reason()}
										onInput={(e) => setReason(e.currentTarget.value)}
										required
									/>
								</div>
								<div class="flex gap-2">
									<Button type="submit" disabled={!canSubmit()}>
										Submit
									</Button>
									<Button type="button" variant="ghost" onClick={close}>
										Cancel
									</Button>
								</div>
							</form>
						)}
					</Show>

					<div class="flex gap-1">
						<For each={KIND_CHIPS}>
							{(chip) => (
								<Button
									variant={kind() === chip.kind ? "default" : "outline"}
									size="sm"
									aria-pressed={kind() === chip.kind}
									onClick={() => pickKind(chip.kind)}
								>
									{chip.label}
								</Button>
							)}
						</For>
					</div>

					<Show when={(feedback.data?.outdatedCount ?? 0) > 0}>
						<label class="flex items-center gap-2 text-xs text-muted">
							<input
								type="checkbox"
								checked={showOutdated()}
								onChange={(e) => {
									setShowOutdated(e.currentTarget.checked);
									setPage(1);
								}}
							/>
							Show {feedback.data?.outdatedCount} outdated
						</label>
					</Show>

					<ul class="flex flex-col gap-2">
						<For
							each={feedback.data?.entries}
							fallback={<li class="text-sm text-faint">No feedback yet.</li>}
						>
							{(entry) => <FeedbackEntry entry={entry} />}
						</For>
					</ul>

					<div class="flex items-center justify-between text-xs text-faint">
						<Button
							variant="ghost"
							size="sm"
							disabled={page() <= 1}
							onClick={() => setPage(page() - 1)}
						>
							Previous
						</Button>
						<span>
							Page {page()} of {lastPage()}
						</span>
						<Button
							variant="ghost"
							size="sm"
							disabled={page() >= lastPage()}
							onClick={() => setPage(page() + 1)}
						>
							Next
						</Button>
					</div>
				</aside>
			</Show>
		</>
	);
}

function jobTarget(composing: Composing): JobTarget | null {
	return composing.kind === "job" ? composing.target : null;
}

function collectionTarget(composing: Composing): CollectionTarget | null {
	return composing.kind === "collection" ? composing.target : null;
}

function CollectionSummary(props: { target: CollectionTarget }): JSX.Element {
	const filters = () => Object.entries(props.target.filters);
	return (
		<div class="flex flex-col gap-1 rounded-md border border-border p-3 text-sm">
			<span class="font-medium text-foreground">
				{props.target.jobIds.length} Jobs, in displayed order
			</span>
			<span class="text-2xs text-faint">
				{filters().length === 0
					? "No filters"
					: filters()
							.map(([k, v]) => `${k}=${v}`)
							.join(", ")}
			</span>
		</div>
	);
}

function JobEvidence(props: { target: JobTarget }): JSX.Element {
	return (
		<div class="flex flex-col gap-1 rounded-md border border-border p-3 text-sm">
			<span class="font-medium text-foreground">{props.target.title}</span>
			<span class="text-muted">Score {props.target.score}</span>
			<ul class="flex flex-wrap gap-1 text-2xs text-faint">
				<For each={props.target.breakdown}>
					{(row) => (
						<li>
							{row.label}: {row.effect}
						</li>
					)}
				</For>
			</ul>
		</div>
	);
}

function FeedbackEntry(props: { entry: ScoreFeedback }): JSX.Element {
	const remove = useDeleteScoreFeedback();
	return (
		<li class="flex flex-col gap-1 rounded-md border border-border p-3">
			<span class="text-2xs tracking-wider text-faint uppercase">
				{props.entry.kind}
				{props.entry.direction ? ` · ${props.entry.direction}` : ""} ·{" "}
				{new Date(props.entry.createdAt).toLocaleString()}
			</span>
			<Show when={props.entry.picksChanged || props.entry.modelChanged}>
				<span class="text-2xs font-medium text-muted">
					{[
						props.entry.picksChanged && "picks changed",
						props.entry.modelChanged && "model changed",
					]
						.filter(Boolean)
						.join(" · ")}
				</span>
			</Show>
			<p class="text-sm whitespace-pre-wrap text-foreground">
				{props.entry.reason}
			</p>
			<Show when={props.entry.snapshot?.options?.length}>
				<details class="text-xs text-muted">
					<summary class="cursor-pointer">
						Score {props.entry.snapshot?.score} · probabilities
					</summary>
					<table class="mt-1 w-full text-left">
						<thead>
							<tr>
								<th>Option</th>
								<th>Yes</th>
								<th>No</th>
								<th>n/s</th>
							</tr>
						</thead>
						<tbody>
							<For each={props.entry.snapshot?.options}>
								{(o) => (
									<tr>
										<td>{o.label}</td>
										<Show
											when={o.known}
											fallback={<td colspan={3}>no cached answer</td>}
										>
											<td>{o.pYes.toFixed(2)}</td>
											<td>{o.pNo.toFixed(2)}</td>
											<td>{o.pNotStated.toFixed(2)}</td>
										</Show>
									</tr>
								)}
							</For>
						</tbody>
					</table>
				</details>
			</Show>
			<Button
				variant="ghost"
				size="sm"
				class="self-end"
				disabled={remove.isPending}
				onClick={() => remove.mutate(props.entry.id)}
			>
				Delete
			</Button>
		</li>
	);
}
