import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { ConfirmDeleteDialog } from "@/components/ConfirmDeleteDialog";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { DEFAULT_STATUS_HEX, STATUS_PALETTE } from "@/lib/status";
import {
	useApplicationStatuses,
	useCreateApplicationStatus,
	useDeleteApplicationStatus,
	useUpdateApplicationStatus,
} from "../../../hooks/useApplicationStatuses";
import type { ApplicationStatus } from "../../../types/applicationStatus";

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
	const [newColour, setNewColour] = createSignal(DEFAULT_STATUS_HEX);

	const [editingId, setEditingId] = createSignal<string | null>(null);
	const [editName, setEditName] = createSignal("");
	const [editColour, setEditColour] = createSignal("");

	const [deletingStatus, setDeletingStatus] =
		createSignal<ApplicationStatus | null>(null);

	const addForm = useFormSubmit(async () => {
		if (!newName().trim()) return;
		await createMutation.mutateAsync({
			name: newName().trim(),
			colour: newColour(),
		});
		setNewName("");
		setNewColour(DEFAULT_STATUS_HEX);
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
		const result = await deleteMutation.mutateAsync(id);
		if (result.count !== undefined) {
			throw new Error(
				`Cannot delete: ${result.count} application${result.count !== 1 ? "s" : ""} use this status. Reassign them first.`,
			);
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
			<FormFeedback error={addForm.error() ?? editForm.error()} />

			<QueryBoundary query={query} fallbackRows={5}>
				{(data) => (
					<>
						<Card class="overflow-hidden divide-y divide-border">
							<For each={data()}>
								{(status) => (
									<form
										onSubmit={editForm.submit}
										class="flex items-center gap-3 px-4 py-3"
									>
										<Show
											when={editingId() === status.ID}
											fallback={
												<>
													<span
														class="size-3 shrink-0 rounded-full"
														style={{ background: status.Colour }}
													/>
													<span class="flex-1 text-sm font-medium text-foreground">
														{status.Name}
													</span>
													<button
														type="button"
														onClick={() => startEdit(status)}
														class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
													>
														Edit
													</button>
													<button
														type="button"
														onClick={() => setDeletingStatus(status)}
														class="rounded px-2 py-1 text-xs font-medium text-destructive-strong transition hover:bg-destructive-subtle"
													>
														Delete
													</button>
												</>
											}
										>
											<div class="flex flex-1 items-center gap-2">
												<div class="flex gap-1">
													<For each={STATUS_PALETTE}>
														{(p) => (
															<button
																type="button"
																title={p.label}
																onClick={() => setEditColour(p.hex)}
																class="h-5 w-5 rounded-full border-2 transition"
																style={{
																	background: p.hex,
																	"border-color":
																		editColour() === p.hex
																			? "var(--color-foreground)"
																			: "transparent",
																}}
															/>
														)}
													</For>
												</div>
												<Input
													class="flex-1"
													aria-label="Status name"
													value={editName()}
													onInput={(e) => setEditName(e.currentTarget.value)}
												/>
											</div>
											<button
												type="submit"
												disabled={editForm.pending()}
												class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
											>
												Save
											</button>
											<button
												type="button"
												onClick={() => setEditingId(null)}
												class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
											>
												Cancel
											</button>
										</Show>
									</form>
								)}
							</For>

							<Show when={showAdd()}>
								<form
									onSubmit={addForm.submit}
									class="flex items-center gap-2 px-4 py-3"
								>
									<div class="flex gap-1">
										<For each={STATUS_PALETTE}>
											{(p) => (
												<button
													type="button"
													title={p.label}
													onClick={() => setNewColour(p.hex)}
													class="h-5 w-5 rounded-full border-2 transition"
													style={{
														background: p.hex,
														"border-color":
															newColour() === p.hex
																? "var(--color-foreground)"
																: "transparent",
													}}
												/>
											)}
										</For>
									</div>
									<Input
										class="flex-1"
										aria-label="Status name"
										placeholder="Status name"
										value={newName()}
										onInput={(e) => setNewName(e.currentTarget.value)}
									/>
									<button
										type="submit"
										disabled={addForm.pending() || !newName().trim()}
										class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
									>
										Add
									</button>
									<button
										type="button"
										onClick={() => {
											setShowAdd(false);
											setNewName("");
											addForm.setError(null);
										}}
										class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
									>
										Cancel
									</button>
								</form>
							</Show>
						</Card>

						<p class="mt-3 text-xs text-faint">
							Changes take effect immediately.
						</p>
					</>
				)}
			</QueryBoundary>
			<ConfirmDeleteDialog
				open={deletingStatus() !== null}
				onClose={() => setDeletingStatus(null)}
				title="Delete status?"
				confirmLabel="Delete status"
				description={
					<>
						This removes the{" "}
						<span class="font-medium text-foreground">
							{deletingStatus()?.Name}
						</span>{" "}
						status. This can't be undone.
					</>
				}
				onConfirm={() => handleDelete(deletingStatus()?.ID ?? "")}
			/>
		</>
	);
}
