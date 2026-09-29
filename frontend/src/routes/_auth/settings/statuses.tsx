import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { STATUS_FALLBACK_COLOUR, STATUS_PALETTE } from "@/lib/status";
import {
	useApplicationStatuses,
	useCreateApplicationStatus,
	useDeleteApplicationStatus,
	useUpdateApplicationStatus,
} from "../../../hooks/useApplicationStatuses";
import type { ApplicationStatus } from "../../../types/applicationStatus";
import { AddStatusForm } from "./-statuses/AddStatusForm";
import { StatusRow } from "./-statuses/StatusRow";

export const Route = createFileRoute("/_auth/settings/statuses")({
	component: StatusesPage,
});

function StatusesPage() {
	const query = useApplicationStatuses();
	const createMutation = useCreateApplicationStatus();
	const updateMutation = useUpdateApplicationStatus();
	const deleteMutation = useDeleteApplicationStatus();

	const [showAdd, setShowAdd] = createSignal(false);
	const [newName, setNewName] = createSignal("");
	const [newColour, setNewColour] = createSignal(
		STATUS_PALETTE[0]?.hex ?? STATUS_FALLBACK_COLOUR,
	);

	const [editingId, setEditingId] = createSignal<string | null>(null);
	const [editName, setEditName] = createSignal("");
	const [editColour, setEditColour] = createSignal("");

	const [deleteError, setDeleteError] = createSignal<string | null>(null);

	const addForm = useFormSubmit(async () => {
		if (!newName().trim()) return;
		await createMutation.mutateAsync({
			name: newName().trim(),
			colour: newColour(),
		});
		setNewName("");
		setNewColour(STATUS_PALETTE[0]?.hex ?? STATUS_FALLBACK_COLOUR);
		setShowAdd(false);
	});

	const startEdit = (s: ApplicationStatus) => {
		editForm.setError(null);
		setEditingId(s.ID);
		setEditName(s.Name);
		setEditColour(s.Colour);
	};

	const editForm = useFormSubmit(async () => {
		const id = editingId();
		if (!id || !editName().trim()) return;
		await updateMutation.mutateAsync({
			id,
			name: editName().trim(),
			colour: editColour(),
		});
		setEditingId(null);
	});

	const handleDelete = async (id: string) => {
		setDeleteError(null);
		try {
			const result = await deleteMutation.mutateAsync(id);
			if (result.count !== undefined) {
				setDeleteError(
					`Cannot delete: ${result.count} application${result.count !== 1 ? "s" : ""} use this status. Reassign them first.`,
				);
			}
		} catch {
			setDeleteError("Could not delete the status. Please try again.");
		}
	};

	return (
		<>
			<SettingsActions>
				<Show when={!showAdd()}>
					<Button size="sm" onClick={() => setShowAdd(true)}>
						<Icon name="plus" size={12} strokeWidth={2.5} />
						Add status
					</Button>
				</Show>
			</SettingsActions>
			<FormFeedback
				error={addForm.error() ?? editForm.error() ?? deleteError()}
			/>

			<QueryBoundary query={query} fallbackRows={5}>
				{(data) => (
					<>
						<Card class="overflow-hidden divide-y divide-border">
							<For each={data()}>
								{(status) => (
									<StatusRow
										status={status}
										editing={editingId() === status.ID}
										name={editName()}
										colour={editColour()}
										pending={editForm.pending()}
										onName={setEditName}
										onColour={setEditColour}
										onEdit={() => startEdit(status)}
										onCancel={() => setEditingId(null)}
										onDelete={() => handleDelete(status.ID)}
										onSubmit={editForm.submit}
									/>
								)}
							</For>

							<Show when={showAdd()}>
								<AddStatusForm
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
		</>
	);
}
