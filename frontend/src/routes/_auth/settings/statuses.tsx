import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { parseReplyWindow } from "@/lib/replyWindow";
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

	const [newReplyWindow, setNewReplyWindow] = createSignal("");

	const [editingId, setEditingId] = createSignal<string | null>(null);
	const [editName, setEditName] = createSignal("");
	const [editColour, setEditColour] = createSignal("");
	const [editReplyWindow, setEditReplyWindow] = createSignal("");

	const [deleteError, setDeleteError] = createSignal<string | null>(null);

	const addForm = useFormSubmit(async () => {
		const window = parseReplyWindow(newReplyWindow());
		if (!newName().trim() || window.error !== undefined) return;
		await createMutation.mutateAsync({
			name: newName().trim(),
			colour: newColour(),
			replyWindowDays: window.days,
		});
		setNewName("");
		setNewReplyWindow("");
		setNewColour(STATUS_PALETTE[0]?.hex ?? STATUS_FALLBACK_COLOUR);
		setShowAdd(false);
	});

	const startEdit = (s: ApplicationStatus) => {
		editForm.setError(null);
		setEditingId(s.ID);
		setEditName(s.Name);
		setEditColour(s.Colour);
		setEditReplyWindow(s.ReplyWindowDays?.toString() ?? "");
	};

	const editForm = useFormSubmit(async () => {
		const id = editingId();
		const window = parseReplyWindow(editReplyWindow());
		if (!id || !editName().trim() || window.error !== undefined) return;
		await updateMutation.mutateAsync({
			id,
			name: editName().trim(),
			colour: editColour(),
			replyWindowDays: window.days,
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
										replyWindow={editReplyWindow()}
										pending={editForm.pending()}
										onName={setEditName}
										onColour={setEditColour}
										onReplyWindow={setEditReplyWindow}
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
									replyWindow={newReplyWindow()}
									pending={addForm.pending()}
									onName={setNewName}
									onColour={setNewColour}
									onReplyWindow={setNewReplyWindow}
									onCancel={() => {
										setShowAdd(false);
										setNewName("");
										setNewReplyWindow("");
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
