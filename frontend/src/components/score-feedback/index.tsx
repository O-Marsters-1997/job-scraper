import { createEffect, createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import {
	useAppendOverallFeedback,
	useDeleteScoreFeedback,
	useScoreFeedback,
} from "@/hooks/useScoreFeedback";
import {
	SCORE_FEEDBACK_PAGE_SIZE,
	type ScoreFeedbackKind,
} from "@/types/scoreFeedback";

const KIND_CHIPS: { label: string; kind?: ScoreFeedbackKind }[] = [
	{ label: "All" },
	{ label: "Jobs", kind: "job" },
	{ label: "Collections", kind: "collection" },
	{ label: "Overall", kind: "overall" },
];

export default function ScoreFeedbackPanel() {
	const [open, setOpen] = createSignal(false);
	const [composing, setComposing] = createSignal(false);
	const [reason, setReason] = createSignal("");
	const [kind, setKind] = createSignal<ScoreFeedbackKind | undefined>();
	const [page, setPage] = createSignal(1);
	const feedback = useScoreFeedback(kind, page);
	const append = useAppendOverallFeedback();
	const remove = useDeleteScoreFeedback();
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

	const form = useFormSubmit(async () => {
		await append.mutateAsync({ reason: reason() });
		setReason("");
		setComposing(false);
	});

	return (
		<>
			<button
				type="button"
				class="fixed right-4 bottom-16 z-40 cursor-pointer rounded-full border border-border bg-surface px-3 py-1.5 text-xs font-medium text-muted shadow-sm hover:border-border-strong hover:text-foreground"
				onClick={() => setOpen(true)}
			>
				Score feedback
			</button>
			<Show when={open()}>
				<aside
					aria-label="Score feedback"
					class="fixed inset-y-0 right-0 z-50 flex w-96 max-w-full flex-col gap-4 border-l border-border bg-surface p-5"
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
							<Button variant="outline" onClick={() => setComposing(true)}>
								Overall
							</Button>
						}
					>
						<form class="flex flex-col gap-3" onSubmit={form.submit}>
							<FormFeedback error={form.error()} />
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
								<Button
									type="submit"
									disabled={form.pending() || reason().trim() === ""}
								>
									Submit
								</Button>
								<Button variant="ghost" onClick={() => setComposing(false)}>
									Cancel
								</Button>
							</div>
						</form>
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

					<ul class="flex flex-col gap-2 overflow-y-auto">
						<For
							each={feedback.data?.entries}
							fallback={<li class="text-sm text-faint">No feedback yet.</li>}
						>
							{(entry) => (
								<li class="flex flex-col gap-1 rounded-md border border-border p-3">
									<span class="text-2xs tracking-wider text-faint uppercase">
										{entry.kind} · {new Date(entry.createdAt).toLocaleString()}
									</span>
									<p class="text-sm whitespace-pre-wrap text-foreground">
										{entry.reason}
									</p>
									<Button
										variant="ghost"
										size="sm"
										class="self-end"
										disabled={remove.isPending}
										onClick={() => remove.mutate(entry.id)}
									>
										Delete
									</Button>
								</li>
							)}
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
