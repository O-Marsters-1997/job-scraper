import { createSignal, For, Show } from "solid-js";
import { ApiError } from "@/api/client";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { useCreateTask, useTaskTypes } from "@/hooks/useTasks";
import type { Job } from "@/types/job";

const DEFAULT_TYPE_NAME = "Apply";

interface CreateTaskDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	job?: Pick<Job, "ID" | "Title"> | undefined;
}

export function CreateTaskDialog(props: CreateTaskDialogProps) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add task</DialogTitle>
					<Show when={props.job}>
						{(job) => <p class="text-sm text-muted">{job().Title}</p>}
					</Show>
				</DialogHeader>
				<Show when={props.open}>
					<CreateTaskForm
						job={props.job}
						onClose={() => props.onOpenChange(false)}
					/>
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function errorMessage(err: unknown): string {
	if (err instanceof ApiError) {
		const body = err.body as { error?: unknown } | undefined;
		if (typeof body?.error === "string") return body.error;
	}
	return "Couldn't save the task. Try again.";
}

function CreateTaskForm(props: {
	job: Pick<Job, "ID" | "Title"> | undefined;
	onClose: () => void;
}) {
	const types = useTaskTypes();
	const create = useCreateTask();
	const [typeId, setTypeId] = createSignal("");
	const [link, setLink] = createSignal("");
	const [notes, setNotes] = createSignal("");

	const selectedTypeId = () =>
		typeId() ||
		types.data?.find((t) => t.Name === DEFAULT_TYPE_NAME)?.ID ||
		types.data?.[0]?.ID ||
		"";

	const form = useFormSubmit(async () => {
		await create.mutateAsync({
			type_id: selectedTypeId(),
			job_id: props.job?.ID ?? null,
			link: link().trim(),
			notes: notes().trim(),
		});
		props.onClose();
	}, errorMessage);

	return (
		<form onSubmit={form.submit} class="flex flex-col gap-4">
			<FormFeedback error={form.error()} />
			<div>
				<Label for="task-type">Type</Label>
				<select
					id="task-type"
					class="field"
					value={selectedTypeId()}
					onChange={(e) => setTypeId(e.currentTarget.value)}
				>
					<For each={types.data}>
						{(t) => <option value={t.ID}>{t.Name}</option>}
					</For>
				</select>
			</div>
			<div>
				<Label for="task-link">Link</Label>
				<Input
					id="task-link"
					type="text"
					placeholder="https://…"
					value={link()}
					onInput={(e) => setLink(e.currentTarget.value)}
				/>
			</div>
			<div>
				<Label for="task-notes">Notes</Label>
				<textarea
					id="task-notes"
					class="field resize-y"
					rows={3}
					value={notes()}
					onInput={(e) => setNotes(e.currentTarget.value)}
				/>
			</div>
			<DialogFooter class="border-t border-border pt-4">
				<Button type="button" variant="outline" onClick={props.onClose}>
					Cancel
				</Button>
				<Button
					type="submit"
					disabled={form.pending() || selectedTypeId() === ""}
				>
					{form.pending() ? "Saving…" : "Save"}
				</Button>
			</DialogFooter>
		</form>
	);
}
