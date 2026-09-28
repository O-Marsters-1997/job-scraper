import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { STATUS_FALLBACK_COLOUR, STATUS_PALETTE } from "@/lib/status";
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
	const [newColour, setNewColour] = createSignal(
		STATUS_PALETTE[0]?.hex ?? STATUS_FALLBACK_COLOUR,
	);

	const [editingId, setEditingId] = createSignal<string | null>(null);
	const [editName, setEditName] = createSignal("");
	const [editColour, setEditColour] = createSignal("");

	const [deleteError, setDeleteError] = createSignal<string | null>(null);

	const handleAdd = async () => {
		if (!newName().trim()) return;
		await createMutation.mutateAsync({
			name: newName().trim(),
			colour: newColour(),
		});
		setNewName("");
		setNewColour(STATUS_PALETTE[0]?.hex ?? STATUS_FALLBACK_COLOUR);
		setShowAdd(false);
	};

	const startEdit = (s: ApplicationStatus) => {
		setEditingId(s.ID);
		setEditName(s.Name);
		setEditColour(s.Colour);
	};

	const handleUpdate = async () => {
		const id = editingId();
		if (!id || !editName().trim()) return;
		await updateMutation.mutateAsync({
			id,
			name: editName().trim(),
			colour: editColour(),
		});
		setEditingId(null);
	};

	const handleDelete = async (id: string) => {
		setDeleteError(null);
		const result = await deleteMutation.mutateAsync(id);
		if (result.count !== undefined) {
			setDeleteError(
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
			<FormFeedback error={deleteError()} />

			<QueryBoundary query={query} fallbackRows={5}>
				{(data) => (
					<>
						<Card class="overflow-hidden divide-y divide-border">
							<For each={data()}>
								{(status) => (
									<div class="flex items-center gap-3 px-4 py-3">
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
														onClick={() => handleDelete(status.ID)}
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
												type="button"
												onClick={handleUpdate}
												disabled={updateMutation.isPending}
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
									</div>
								)}
							</For>

							<Show when={showAdd()}>
								<div class="flex items-center gap-2 px-4 py-3">
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
										onKeyDown={(e) => e.key === "Enter" && handleAdd()}
									/>
									<button
										type="button"
										onClick={handleAdd}
										disabled={createMutation.isPending || !newName().trim()}
										class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
									>
										Add
									</button>
									<button
										type="button"
										onClick={() => {
											setShowAdd(false);
											setNewName("");
										}}
										class="rounded px-2 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
									>
										Cancel
									</button>
								</div>
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
