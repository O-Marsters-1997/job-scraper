import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import {
	taskTypesQueryOptions,
	useCreateTaskType,
	useTaskTypes,
	useUpdateTaskType,
} from "@/hooks/useTasks";
import { queryClient } from "@/lib/queryClient";
import { STATUS_FALLBACK_COLOUR, STATUS_PALETTE } from "@/lib/status";
import { type TaskType, TaskTypeNameTakenError } from "@/types/task";
import { AddTaskTypeForm } from "./-task-types/AddTaskTypeForm";
import { DeleteTaskTypeDialog } from "./-task-types/DeleteTaskTypeDialog";
import { TaskTypeRow } from "./-task-types/TaskTypeRow";

export const Route = createFileRoute("/_auth/settings/task-types")({
	loader: () => queryClient.prefetchQuery(taskTypesQueryOptions),
	component: TaskTypesPage,
});

const DEFAULT_COLOUR = STATUS_PALETTE[0]?.hex ?? STATUS_FALLBACK_COLOUR;

function saveError(err: unknown): string {
	return err instanceof TaskTypeNameTakenError
		? err.message
		: "Could not save the task type. Please try again.";
}

function TaskTypesPage() {
	const query = useTaskTypes();
	const createMutation = useCreateTaskType();
	const updateMutation = useUpdateTaskType();

	const [showAdd, setShowAdd] = createSignal(false);
	const [newName, setNewName] = createSignal("");
	const [newColour, setNewColour] = createSignal(DEFAULT_COLOUR);

	const [editingId, setEditingId] = createSignal<string | null>(null);
	const [editName, setEditName] = createSignal("");
	const [editColour, setEditColour] = createSignal("");

	const [deleting, setDeleting] = createSignal<TaskType | null>(null);

	const addForm = useFormSubmit(async () => {
		if (!newName().trim()) return;
		await createMutation.mutateAsync({
			name: newName().trim(),
			colour: newColour(),
		});
		setNewName("");
		setNewColour(DEFAULT_COLOUR);
		setShowAdd(false);
	}, saveError);

	const editForm = useFormSubmit(async () => {
		const id = editingId();
		if (!id || !editName().trim()) return;
		await updateMutation.mutateAsync({
			id,
			name: editName().trim(),
			colour: editColour(),
		});
		setEditingId(null);
	}, saveError);

	const startEdit = (t: TaskType) => {
		editForm.setError(null);
		setEditingId(t.ID);
		setEditName(t.Name);
		setEditColour(t.Colour);
	};

	return (
		<>
			<SettingsActions>
				<Show when={!showAdd()}>
					<Button size="sm" onClick={() => setShowAdd(true)}>
						<Icon name="plus" size={12} strokeWidth={2.5} />
						Add task type
					</Button>
				</Show>
			</SettingsActions>
			<FormFeedback error={addForm.error() ?? editForm.error()} />

			<QueryBoundary query={query} fallbackRows={4}>
				{(data) => (
					<>
						<Card class="overflow-hidden divide-y divide-border">
							<For each={data()}>
								{(type) => (
									<TaskTypeRow
										type={type}
										editing={editingId() === type.ID}
										name={editName()}
										colour={editColour()}
										pending={editForm.pending()}
										onName={setEditName}
										onColour={setEditColour}
										onEdit={() => startEdit(type)}
										onCancel={() => setEditingId(null)}
										onDelete={() => setDeleting(type)}
										onSubmit={editForm.submit}
									/>
								)}
							</For>

							<Show when={showAdd()}>
								<AddTaskTypeForm
									name={newName()}
									colour={newColour()}
									pending={addForm.pending()}
									onName={setNewName}
									onColour={setNewColour}
									onCancel={() => {
										setShowAdd(false);
										setNewName("");
										addForm.setError(null);
									}}
									onSubmit={addForm.submit}
								/>
							</Show>
						</Card>

						<p class="mt-3 text-xs text-faint">
							Changes take effect immediately.
						</p>
					</>
				)}
			</QueryBoundary>

			<DeleteTaskTypeDialog
				type={deleting()}
				onClose={() => setDeleting(null)}
			/>
		</>
	);
}
