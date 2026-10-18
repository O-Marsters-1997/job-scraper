import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
	useCreateJobNote,
	useDeleteJobNote,
	useJobNotes,
	useUpdateJobNote,
} from "@/hooks/useJobNotes";
import { formatDate } from "@/lib/datetime";
import { loadedData } from "@/lib/queryData";
import { JOB_NOTE_MAX_LENGTH, type JobNote } from "@/types/jobNote";

function NoteItem(props: { note: JobNote; jobId: string }) {
	const update = useUpdateJobNote(() => props.jobId);
	const remove = useDeleteJobNote(() => props.jobId);
	const [editing, setEditing] = createSignal(false);
	const [draft, setDraft] = createSignal("");

	const startEdit = () => {
		setDraft(props.note.Body);
		setEditing(true);
	};
	const save = (e: SubmitEvent) => {
		e.preventDefault();
		update.mutate(
			{ id: props.note.ID, body: draft() },
			{ onSuccess: () => setEditing(false) },
		);
	};

	return (
		<li id={`job-note-${props.note.ID}`} class="flex flex-col gap-0.5">
			<span class="font-mono text-xs tabular-nums text-muted">
				{formatDate(props.note.CreatedAt)}
				<Show when={props.note.UpdatedAt !== props.note.CreatedAt}>
					{" "}
					(edited)
				</Show>
			</span>
			<Show
				when={editing()}
				fallback={
					<>
						<p class="whitespace-pre-wrap break-words text-sm text-foreground">
							{props.note.Body}
						</p>
						<FormFeedback
							error={remove.isError ? "Could not delete the note." : null}
						/>
						<div class="flex gap-2">
							<Button
								type="button"
								size="sm"
								variant="ghost"
								aria-label={`Edit note from ${formatDate(props.note.CreatedAt)}`}
								onClick={startEdit}
							>
								Edit
							</Button>
							<Button
								type="button"
								size="sm"
								variant="ghost"
								aria-label={`Delete note from ${formatDate(props.note.CreatedAt)}`}
								disabled={remove.isPending}
								onClick={() => remove.mutate(props.note.ID)}
							>
								Delete
							</Button>
						</div>
					</>
				}
			>
				<form onSubmit={save} class="flex flex-col gap-2">
					<FormFeedback
						error={update.isError ? "Could not save the note." : null}
					/>
					<Label for={`job-note-edit-${props.note.ID}`}>Edit note</Label>
					<Textarea
						id={`job-note-edit-${props.note.ID}`}
						rows={3}
						maxLength={JOB_NOTE_MAX_LENGTH}
						value={draft()}
						onInput={(e) => setDraft(e.currentTarget.value)}
					/>
					<div class="flex gap-2">
						<Button
							type="submit"
							size="sm"
							disabled={update.isPending || draft().length === 0}
						>
							Save
						</Button>
						<Button
							type="button"
							size="sm"
							variant="ghost"
							onClick={() => setEditing(false)}
						>
							Cancel
						</Button>
					</div>
				</form>
			</Show>
		</li>
	);
}

export function JobNotesCard(props: { jobId: string }) {
	const notes = useJobNotes(() => props.jobId);
	const create = useCreateJobNote(() => props.jobId);
	const [body, setBody] = createSignal("");

	const submit = (e: SubmitEvent) => {
		e.preventDefault();
		create.mutate(body(), { onSuccess: () => setBody("") });
	};

	return (
		<Card>
			<CardContent class="gap-3 pt-4">
				<form onSubmit={submit} class="flex flex-col gap-2">
					<FormFeedback
						error={create.isError ? "Could not save the note." : null}
					/>
					<Label for="job-note-body">Add note</Label>
					<Textarea
						id="job-note-body"
						rows={14}
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
				<Show when={(loadedData(notes)?.length ?? 0) > 0}>
					<ul class="flex flex-col gap-3">
						<For each={loadedData(notes)}>
							{(n) => <NoteItem note={n} jobId={props.jobId} />}
						</For>
					</ul>
				</Show>
			</CardContent>
		</Card>
	);
}
