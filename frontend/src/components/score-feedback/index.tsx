import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import {
	useAppendOverallFeedback,
	useScoreFeedback,
} from "@/hooks/useScoreFeedback";

export default function ScoreFeedbackPanel() {
	const [open, setOpen] = createSignal(false);
	const [composing, setComposing] = createSignal(false);
	const [reason, setReason] = createSignal("");
	const feedback = useScoreFeedback();
	const append = useAppendOverallFeedback();

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

					<ul class="flex flex-col gap-2 overflow-y-auto">
						<For
							each={feedback.data}
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
								</li>
							)}
						</For>
					</ul>
				</aside>
			</Show>
		</>
	);
}
