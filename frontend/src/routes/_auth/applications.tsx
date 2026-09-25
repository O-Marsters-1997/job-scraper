import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import { cn } from "@/lib/utils";
import { StatusBadge } from "../../components/StatusBadge";
import { useApplicationStatuses } from "../../hooks/useApplicationStatuses";
import {
	useApplications,
	useDeleteApplication,
	useUpdateApplication,
} from "../../hooks/useApplications";
import type { ApplicationWithDetails } from "../../types/application";

export const Route = createFileRoute("/_auth/applications")({
	validateSearch: (search: Record<string, unknown>) => ({
		status: typeof search.status === "string" ? search.status : undefined,
	}),
	component: ApplicationsPage,
});

function ApplicationsPage() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const query = useApplications(() => search().status);
	const statusesQuery = useApplicationStatuses();
	const updateMutation = useUpdateApplication();
	const deleteMutation = useDeleteApplication();

	const [modalOpen, setModalOpen] = createSignal(false);
	const [editingApp, setEditingApp] =
		createSignal<ApplicationWithDetails | null>(null);
	const [editStatusId, setEditStatusId] = createSignal("");
	const [editNotes, setEditNotes] = createSignal("");
	const [editAppliedAt, setEditAppliedAt] = createSignal("");
	const [editSalary, setEditSalary] = createSignal("");

	const openEdit = (app: ApplicationWithDetails) => {
		setEditingApp(app);
		setEditStatusId(app.StatusID);
		setEditNotes(app.Notes);
		setEditAppliedAt(app.AppliedAt ?? "");
		setEditSalary(app.SalaryInfo);
		setModalOpen(true);
	};

	const closeEdit = () => setModalOpen(false);

	const [deletingApp, setDeletingApp] =
		createSignal<ApplicationWithDetails | null>(null);

	const handleSave = async () => {
		const app = editingApp();
		if (!app) return;
		await updateMutation.mutateAsync({
			id: app.ID,
			data: {
				status_id: editStatusId() || undefined,
				notes: editNotes(),
				applied_at: editAppliedAt() || null,
				salary_info: editSalary(),
			},
		});
		closeEdit();
	};

	const confirmDelete = async () => {
		const app = deletingApp();
		if (!app) return;
		await deleteMutation.mutateAsync(app.ID);
		setDeletingApp(null);
	};

	const statusColour = (app: ApplicationWithDetails) => {
		if (app.StatusColour) return app.StatusColour;
		const status = statusesQuery.data?.find((s) => s.ID === app.StatusID);
		return status?.Colour ?? STATUS_FALLBACK_COLOUR;
	};

	const chipClass = (active: boolean) =>
		cn(
			"inline-flex cursor-pointer items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition",
			active
				? "border-primary bg-accent-subtle text-accent-text"
				: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
		);

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Applications"
				subtitle="Track where each application stands"
			/>

			<div class="mb-4 flex flex-wrap items-center gap-1.5">
				<button
					type="button"
					onClick={() =>
						navigate({ to: "/applications", search: { status: undefined } })
					}
					class={chipClass(!search().status)}
				>
					All
				</button>
				<For each={statusesQuery.data}>
					{(s) => (
						<button
							type="button"
							onClick={() =>
								navigate({ to: "/applications", search: { status: s.ID } })
							}
							class={chipClass(search().status === s.ID)}
						>
							<span
								class="inline-block size-2 rounded-full"
								style={{ background: s.Colour }}
							/>
							{s.Name}
						</button>
					)}
				</For>
			</div>

			<QueryBoundary query={query} fallbackRows={5}>
				{(data) => (
					<Show
						when={data.length > 0}
						fallback={
							<Card class="p-10 text-center">
								<p class="text-sm text-muted">
									No applications yet. Track a job from the Jobs page.
								</p>
							</Card>
						}
					>
						<div class="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface">
							<For each={data}>
								{(app) => (
									<div class="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-surface-muted">
										<div class="min-w-0 flex-1">
											<p class="truncate text-sm font-medium text-foreground">
												{app.JobTitle}
											</p>
											<p class="text-xs text-faint">
												{app.JobCompanySlug}
												{app.JobLocation ? ` · ${app.JobLocation}` : ""}
											</p>
										</div>
										<Show when={app.StatusName}>
											<StatusBadge
												name={app.StatusName}
												colour={statusColour(app)}
											/>
										</Show>
										<Show when={app.AppliedAt}>
											<span class="shrink-0 font-mono text-xs tabular-nums text-faint">
												{app.AppliedAt}
											</span>
										</Show>
										<button
											type="button"
											onClick={() => openEdit(app)}
											class="shrink-0 rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle"
										>
											Edit
										</button>
										<button
											type="button"
											onClick={() => setDeletingApp(app)}
											class="shrink-0 rounded px-2 py-1 text-xs font-medium text-destructive transition hover:bg-destructive-subtle"
										>
											Delete
										</button>
									</div>
								)}
							</For>
						</div>
					</Show>
				)}
			</QueryBoundary>

			<Dialog open={modalOpen()} onOpenChange={setModalOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Edit application</DialogTitle>
						<p class="text-sm text-faint">{editingApp()?.JobTitle}</p>
					</DialogHeader>

					<div class="flex flex-col gap-4">
						<div>
							<Label for="edit-app-status">Status</Label>
							<select
								id="edit-app-status"
								class="field"
								value={editStatusId()}
								onChange={(e) => setEditStatusId(e.currentTarget.value)}
							>
								<option value="">— No status —</option>
								<For each={statusesQuery.data}>
									{(s) => <option value={s.ID}>{s.Name}</option>}
								</For>
							</select>
						</div>

						<div class="grid grid-cols-2 gap-3">
							<div>
								<Label for="edit-app-applied-at">Applied date</Label>
								<Input
									id="edit-app-applied-at"
									type="date"
									value={editAppliedAt()}
									onInput={(e) => setEditAppliedAt(e.currentTarget.value)}
								/>
							</div>

							<div>
								<Label for="edit-app-salary">Salary / comp</Label>
								<Input
									id="edit-app-salary"
									type="text"
									placeholder="e.g. £80,000"
									value={editSalary()}
									onInput={(e) => setEditSalary(e.currentTarget.value)}
								/>
							</div>
						</div>

						<div>
							<Label for="edit-app-notes">Notes</Label>
							<textarea
								id="edit-app-notes"
								class="field resize-y"
								rows={3}
								placeholder="Any notes about this application…"
								value={editNotes()}
								onInput={(e) => setEditNotes(e.currentTarget.value)}
							/>
						</div>
					</div>

					<DialogFooter class="border-t border-border pt-4">
						<Button variant="outline" onClick={closeEdit}>
							Cancel
						</Button>
						<Button onClick={handleSave} disabled={updateMutation.isPending}>
							Save
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			<Dialog
				open={deletingApp() !== null}
				onOpenChange={(open) => !open && setDeletingApp(null)}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete application?</DialogTitle>
						<p class="text-sm text-muted">
							This removes tracking for{" "}
							<span class="font-medium text-foreground">
								{deletingApp()?.JobTitle}
							</span>
							. The job stays on your Jobs list. This can't be undone.
						</p>
					</DialogHeader>
					<DialogFooter class="border-t border-border pt-4">
						<Button variant="outline" onClick={() => setDeletingApp(null)}>
							Cancel
						</Button>
						<Button
							variant="destructive"
							onClick={confirmDelete}
							disabled={deleteMutation.isPending}
						>
							{deleteMutation.isPending ? "Deleting…" : "Delete application"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
