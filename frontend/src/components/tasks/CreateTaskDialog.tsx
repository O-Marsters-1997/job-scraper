import { createSignal, For, Show, untrack } from "solid-js";
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
import { useCreateTask, useTaskTypes, useUpdateTask } from "@/hooks/useTasks";
import type { Job } from "@/types/job";
import type { Task } from "@/types/task";

const DEFAULT_TYPE_NAME = "Apply";

interface CreateTaskDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	job?: Pick<Job, "ID" | "Title"> | undefined;
	task?: Task | undefined;
}

export function CreateTaskDialog(props: CreateTaskDialogProps) {
	const jobTitle = () => props.task?.JobTitle || props.job?.Title;
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{props.task ? "Edit task" : "Add task"}</DialogTitle>
					<Show when={jobTitle()}>
						{(title) => <p class="text-sm text-muted">{title()}</p>}
					</Show>
				</DialogHeader>
				<Show when={props.open}>
					<CreateTaskForm
						job={props.job}
						task={props.task}
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
	task: Task | undefined;
	onClose: () => void;
}) {
	const types = useTaskTypes();
	const create = useCreateTask();
	const update = useUpdateTask();
	const initial = untrack(() => props.task);
	const [typeId, setTypeId] = createSignal(initial?.TypeID ?? "");
	const [link, setLink] = createSignal(initial?.Link ?? "");
	const [notes, setNotes] = createSignal(initial?.Notes ?? "");

	const selectedTypeId = () =>
		typeId() ||
		types.data?.find((t) => t.Name === DEFAULT_TYPE_NAME)?.ID ||
		types.data?.[0]?.ID ||
		"";

	const form = useFormSubmit(async () => {
		const fields = {
			type_id: selectedTypeId(),
			link: link().trim(),
			notes: notes().trim(),
		};
		if (props.task) {
			await update.mutateAsync({ id: props.task.ID, ...fields });
		} else {
			await create.mutateAsync({ ...fields, job_id: props.job?.ID ?? null });
		}
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
					loading={form.pending()}
					disabled={selectedTypeId() === ""}
				>
					Save
				</Button>
			</DialogFooter>
		</form>
	);
}
