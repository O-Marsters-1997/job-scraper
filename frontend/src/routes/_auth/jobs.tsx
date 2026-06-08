import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { queryClient } from "../../lib/queryClient";
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs";
import {
	useApplicationsForJobs,
	useCreateApplication,
	useUpdateApplication,
} from "../../hooks/useApplications";
import { useApplicationStatuses } from "../../hooks/useApplicationStatuses";
import { createJobColumns } from "../../components/jobs/columns";
import { JobsDataTable } from "../../components/jobs/JobsDataTable";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/_auth/jobs")({
	loader: () => queryClient.ensureQueryData(jobsQueryOptions),
	component: JobsPage,
});

function JobsPage() {
	const query = useJobs();
	const statusesQuery = useApplicationStatuses();

	const jobs = () => query.data ?? [];
	const allJobIds = () => jobs().map((j) => j.ID);

	const appsForJobs = useApplicationsForJobs(allJobIds);
	const createMutation = useCreateApplication();
	const updateMutation = useUpdateApplication();

	const [modalOpen, setModalOpen] = createSignal(false);
	const [trackingJobId, setTrackingJobId] = createSignal<string | null>(null);
	const [modalMode, setModalMode] = createSignal<"create" | "edit">("create");
	const [modalStatusId, setModalStatusId] = createSignal("");
	const [modalNotes, setModalNotes] = createSignal("");
	const [modalAppliedAt, setModalAppliedAt] = createSignal("");
	const [modalSalary, setModalSalary] = createSignal("");

	const openTrack = (jobId: string) => {
		setTrackingJobId(jobId);
		setModalMode("create");
		setModalStatusId("");
		setModalNotes("");
		setModalAppliedAt("");
		setModalSalary("");
		setModalOpen(true);
	};

	const openEdit = (jobId: string) => {
		const app = appsForJobs.data?.[jobId];
		if (!app) return;
		setTrackingJobId(jobId);
		setModalMode("edit");
		setModalStatusId(app.StatusID);
		setModalNotes("");
		setModalAppliedAt("");
		setModalSalary("");
		setModalOpen(true);
	};

	const closeModal = () => setModalOpen(false);

	const handleSubmit = async () => {
		const jobId = trackingJobId();
		if (!jobId) return;
		if (modalMode() === "create") {
			await createMutation.mutateAsync({
				job_id: jobId,
				status_id: modalStatusId() || undefined,
				notes: modalNotes(),
				applied_at: modalAppliedAt() || null,
				salary_info: modalSalary(),
			});
		} else {
			const app = appsForJobs.data?.[jobId];
			if (!app) return;
			await updateMutation.mutateAsync({
				id: app.ApplicationID,
				data: {
					status_id: modalStatusId() || undefined,
					notes: modalNotes(),
					applied_at: modalAppliedAt() || null,
					salary_info: modalSalary(),
				},
			});
		}
		closeModal();
	};

	const columns = createJobColumns({
		appsForJobs: () => appsForJobs.data,
		onTrack: openTrack,
		onEdit: openEdit,
	});

	const currentJob = () =>
		jobs().find((j) => j.ID === trackingJobId());

	return (
		<div class="px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">Jobs</h1>
				<p class="mt-0.5 text-xs text-faint">
					Open roles scraped from your configured sources
				</p>
			</div>

			<Show when={query.isPending}>
				<p class="text-sm text-muted">Loading jobs…</p>
			</Show>

			<Show when={query.isError}>
				<div class="rounded-xl border border-destructive/30 bg-destructive-subtle p-6">
					<p class="mb-1 text-sm font-semibold text-destructive-strong">
						Error
					</p>
					<p class="text-sm text-muted">{query.error?.message}</p>
				</div>
			</Show>

			<Show when={query.isSuccess}>
				<JobsDataTable columns={columns} data={jobs()} />
			</Show>

			{/* Track / Edit modal */}
			<Dialog open={modalOpen()} onOpenChange={setModalOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>
							{modalMode() === "create"
								? "Track application"
								: "Edit application"}
						</DialogTitle>
						<p class="text-sm text-faint">{currentJob()?.Title}</p>
					</DialogHeader>

					<div class="flex flex-col gap-4">
						<div>
							<Label class="field-label">Status</Label>
							<select
								class="field"
								value={modalStatusId()}
								onChange={(e) => setModalStatusId(e.currentTarget.value)}
							>
								<option value="">— No status —</option>
								<For each={statusesQuery.data}>
									{(s) => <option value={s.ID}>{s.Name}</option>}
								</For>
							</select>
						</div>

						<div class="grid grid-cols-2 gap-3">
							<div>
								<Label class="field-label">Applied date</Label>
								<Input
									type="date"
									value={modalAppliedAt()}
									onInput={(e) => setModalAppliedAt(e.currentTarget.value)}
								/>
							</div>

							<div>
								<Label class="field-label">Salary / comp</Label>
								<Input
									type="text"
									placeholder="e.g. £80,000"
									value={modalSalary()}
									onInput={(e) => setModalSalary(e.currentTarget.value)}
								/>
							</div>
						</div>

						<div>
							<Label class="field-label">Notes</Label>
							<textarea
								class="field resize-y"
								rows={3}
								placeholder="Any notes…"
								value={modalNotes()}
								onInput={(e) => setModalNotes(e.currentTarget.value)}
							/>
						</div>
					</div>

					<DialogFooter class="border-t border-border pt-4">
						<Button variant="outline" onClick={closeModal}>
							Cancel
						</Button>
						<Button
							onClick={handleSubmit}
							disabled={
								createMutation.isPending || updateMutation.isPending
							}
						>
							{modalMode() === "create" ? "Save" : "Update"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
