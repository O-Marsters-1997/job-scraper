import { createEffect, createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCreateJobNote, useJobNotes } from "@/hooks/useJobNotes";
import { formatDate } from "@/lib/datetime";
import { JOB_NOTE_MAX_LENGTH, type NoteQuote } from "@/types/jobNote";

function Quote(props: { text: string }) {
	return (
		<blockquote class="line-clamp-4 whitespace-pre-wrap break-words border-l-2 border-border pl-3 text-sm text-muted">
			{props.text}
		</blockquote>
	);
}

export function JobNotesCard(props: {
	jobId: string;
	quote: NoteQuote | null;
	onClearQuote: () => void;
}) {
	const notes = useJobNotes(() => props.jobId);
	const create = useCreateJobNote(() => props.jobId);
	const [body, setBody] = createSignal("");
	let input!: HTMLTextAreaElement;

	createEffect(() => {
		if (props.quote) input.focus();
	});

	const submit = (e: SubmitEvent) => {
		e.preventDefault();
		create.mutate(
			{ body: body(), quote: props.quote },
			{
				onSuccess: () => {
					setBody("");
					props.onClearQuote();
				},
			},
		);
	};

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Notes</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<form onSubmit={submit} class="flex flex-col gap-2">
					<FormFeedback
						error={create.isError ? "Could not save the note." : null}
					/>
					<Show when={props.quote}>
						{(q) => (
							<div class="flex items-start justify-between gap-2">
								<Quote text={q().exact} />
								<Button
									type="button"
									size="sm"
									variant="ghost"
									onClick={props.onClearQuote}
								>
									Remove quote
								</Button>
							</div>
						)}
					</Show>
					<Label for="job-note-body">Add note</Label>
					<Textarea
						id="job-note-body"
						ref={input}
						rows={3}
						maxLength={JOB_NOTE_MAX_LENGTH}
						value={body()}
						onInput={(e) => setBody(e.currentTarget.value)}
					/>
					<Button
						type="submit"
						size="sm"
						disabled={create.isPending || body().length === 0}
					>
						Save note
					</Button>
				</form>
				<Show when={(notes.data?.length ?? 0) > 0}>
					<ul class="flex flex-col gap-3">
						<For each={notes.data}>
							{(n) => (
								<li class="flex flex-col gap-0.5">
									<span class="font-mono text-xs tabular-nums text-muted">
										{formatDate(n.CreatedAt)}
									</span>
									<Show when={n.Quote}>
										{(q) => <Quote text={q().exact} />}
									</Show>
									<p class="whitespace-pre-wrap break-words text-sm text-foreground">
										{n.Body}
									</p>
								</li>
							)}
						</For>
					</ul>
				</Show>
			</CardContent>
		</Card>
	);
}
