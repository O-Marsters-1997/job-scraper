import { createSignal, onCleanup, Show, untrack } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useJobNote, useSaveJobNote } from "@/hooks/useJobNotes";
import { JOB_NOTE_MAX_LENGTH } from "@/types/jobNote";

const AUTOSAVE_DELAY_MS = 800;

function NoteEditor(props: { jobId: string; initial: string }) {
	const save = useSaveJobNote(() => props.jobId);
	const [body, setBody] = createSignal(props.initial);
	const [saved, setSaved] = createSignal<string | undefined>(props.initial);
	let timer: ReturnType<typeof setTimeout> | undefined;

	const flush = () => {
		clearTimeout(timer);
		const text = body();
		if (text === saved()) return;
		setSaved(text);
		save.mutateAsync(text).catch(() => {
			if (untrack(saved) === text) setSaved(undefined);
		});
	};
	onCleanup(flush);

	const onInput = (value: string) => {
		setBody(value);
		clearTimeout(timer);
		timer = setTimeout(flush, AUTOSAVE_DELAY_MS);
	};

	return (
		<div class="flex flex-col gap-2">
			<FormFeedback
				error={
					save.isError
						? "Could not save the note. Your text is kept here."
						: null
				}
			/>
			<Label for="job-note-body">Note</Label>
			<Textarea
				id="job-note-body"
				rows={14}
				maxLength={JOB_NOTE_MAX_LENGTH}
				value={body()}
				onInput={(e) => onInput(e.currentTarget.value)}
				onBlur={flush}
			/>
			<p class="h-4 text-xs text-muted" aria-live="polite">
				<Show when={save.isPending}>Saving…</Show>
				<Show when={save.isSuccess && body() === saved()}>Saved</Show>
			</p>
		</div>
	);
}

export function JobNotesCard(props: { jobId: string }) {
	const note = useJobNote(() => props.jobId);
	return (
		<Card>
			<CardContent class="gap-3 pt-4">
				<QueryBoundary query={note} fallbackRows={4}>
					{(n) => <NoteEditor jobId={props.jobId} initial={n().Body} />}
				</QueryBoundary>
			</CardContent>
		</Card>
	);
}
