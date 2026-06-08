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
	};

	const closeModal = () => setTrackingJobId(null);

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
			<Show when={trackingJobId()}>
				{(jobId) => {
					const job = () => jobs().find((j) => j.ID === jobId());
					return (
						<div
							class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4 backdrop-blur-[2px]"
							onClick={(e) => e.target === e.currentTarget && closeModal()}
						>
							<div class="w-full max-w-md rounded-2xl border border-border bg-surface p-6 shadow-2xl">
								<h2 class="text-base font-semibold text-foreground">
									{modalMode() === "create"
										? "Track application"
										: "Edit application"}
								</h2>
								<p class="mb-5 text-sm text-faint">{job()?.Title}</p>

								<div class="space-y-4">
									<div>
										<label class="field-label">Status</label>
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
											<label class="field-label">Applied date</label>
											<input
												type="date"
												class="field"
												value={modalAppliedAt()}
												onInput={(e) =>
													setModalAppliedAt(e.currentTarget.value)
												}
											/>
										</div>

										<div>
											<label class="field-label">Salary / comp</label>
											<input
												type="text"
												class="field"
												placeholder="e.g. £80,000"
												value={modalSalary()}
												onInput={(e) => setModalSalary(e.currentTarget.value)}
											/>
										</div>
									</div>

									<div>
										<label class="field-label">Notes</label>
										<textarea
											class="field resize-y"
											rows={3}
											placeholder="Any notes…"
											value={modalNotes()}
											onInput={(e) => setModalNotes(e.currentTarget.value)}
										/>
									</div>
								</div>

								<div class="mt-5 flex justify-end gap-2 border-t border-border pt-4">
									<button
										type="button"
										onClick={closeModal}
										class="rounded-md border border-border bg-surface px-4 py-1.5 text-sm font-medium text-muted transition hover:border-border-strong hover:text-foreground"
									>
										Cancel
									</button>
									<button
										type="button"
										onClick={handleSubmit}
										disabled={
											createMutation.isPending || updateMutation.isPending
										}
										class="rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover disabled:opacity-50"
									>
										{modalMode() === "create" ? "Save" : "Update"}
									</button>
								</div>
							</div>
						</div>
					);
				}}
			</Show>
		</div>
	);
}
