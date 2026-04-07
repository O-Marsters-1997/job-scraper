import { createFileRoute } from "@tanstack/solid-router"
import { createSignal, For, Show } from "solid-js"
import { queryClient } from "../../lib/queryClient"
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs"
import { useApplicationsForJobs, useCreateApplication, useUpdateApplication } from "../../hooks/useApplications"
import { useApplicationStatuses } from "../../hooks/useApplicationStatuses"
import { createJobColumns } from "../../components/jobs/columns"
import { JobsDataTable } from "../../components/jobs/JobsDataTable"

export const Route = createFileRoute("/_auth/jobs")({
	loader: () => queryClient.ensureQueryData(jobsQueryOptions),
	component: JobsPage,
})

function JobsPage() {
	const query = useJobs()
	const statusesQuery = useApplicationStatuses()

	const jobs = () => query.data ?? []
	const allJobIds = () => jobs().map((j) => j.ID)

	const appsForJobs = useApplicationsForJobs(allJobIds)
	const createMutation = useCreateApplication()
	const updateMutation = useUpdateApplication()

	const [trackingJobId, setTrackingJobId] = createSignal<string | null>(null)
	const [modalMode, setModalMode] = createSignal<"create" | "edit">("create")
	const [modalStatusId, setModalStatusId] = createSignal("")
	const [modalNotes, setModalNotes] = createSignal("")
	const [modalAppliedAt, setModalAppliedAt] = createSignal("")
	const [modalSalary, setModalSalary] = createSignal("")

	const openTrack = (jobId: string) => {
		setTrackingJobId(jobId)
		setModalMode("create")
		setModalStatusId("")
		setModalNotes("")
		setModalAppliedAt("")
		setModalSalary("")
	}

	const openEdit = (jobId: string) => {
		const app = appsForJobs.data?.[jobId]
		if (!app) return
		setTrackingJobId(jobId)
		setModalMode("edit")
		setModalStatusId(app.StatusID)
		setModalNotes("")
		setModalAppliedAt("")
		setModalSalary("")
	}

	const closeModal = () => setTrackingJobId(null)

	const handleSubmit = async () => {
		const jobId = trackingJobId()
		if (!jobId) return
		if (modalMode() === "create") {
			await createMutation.mutateAsync({
				job_id: jobId,
				status_id: modalStatusId() || undefined,
				notes: modalNotes(),
				applied_at: modalAppliedAt() || null,
				salary_info: modalSalary(),
			})
		} else {
			const app = appsForJobs.data?.[jobId]
			if (!app) return
			await updateMutation.mutateAsync({
				id: app.ApplicationID,
				data: {
					status_id: modalStatusId() || undefined,
					notes: modalNotes(),
					applied_at: modalAppliedAt() || null,
					salary_info: modalSalary(),
				},
			})
		}
		closeModal()
	}

	const columns = createJobColumns({
		appsForJobs: () => appsForJobs.data,
		onTrack: openTrack,
		onEdit: openEdit,
	})

	return (
		<div class="px-6 pb-12 pt-8">
			<div class="mb-8">
				<p class="island-kicker mb-2">Live Listings</p>
				<h1 class="display-title text-4xl font-bold text-[var(--sea-ink)] sm:text-5xl">
					Open Roles
				</h1>
			</div>

			<Show when={query.isPending}>
				<p class="text-sm text-[var(--sea-ink-soft)]">Loading jobs…</p>
			</Show>

			<Show when={query.isError}>
				<div class="island-shell rounded-xl p-6">
					<p class="island-kicker mb-2 text-red-600">Error</p>
					<p class="text-sm text-[var(--sea-ink-soft)]">{query.error?.message}</p>
				</div>
			</Show>

			<Show when={query.isSuccess}>
				<JobsDataTable columns={columns} data={jobs()} />
			</Show>

			{/* Track / Edit modal */}
			<Show when={trackingJobId()}>
				{(jobId) => {
					const job = () => jobs().find((j) => j.ID === jobId())
					return (
						<div
							class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
							onClick={(e) => e.target === e.currentTarget && closeModal()}
						>
							<div class="island-shell w-full max-w-md rounded-2xl p-6">
								<h2 class="mb-1 text-base font-semibold text-[var(--sea-ink)]">
									{modalMode() === "create" ? "Track application" : "Edit application"}
								</h2>
								<p class="mb-4 text-sm text-[var(--sea-ink-soft)]">{job()?.Title}</p>

								<div class="space-y-4">
									<div>
										<label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Status</label>
										<select
											class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none"
											value={modalStatusId()}
											onChange={(e) => setModalStatusId(e.currentTarget.value)}
										>
											<option value="">— No status —</option>
											<For each={statusesQuery.data}>
												{(s) => <option value={s.ID}>{s.Name}</option>}
											</For>
										</select>
									</div>

									<div>
										<label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Applied date</label>
										<input
											type="date"
											class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none"
											value={modalAppliedAt()}
											onInput={(e) => setModalAppliedAt(e.currentTarget.value)}
										/>
									</div>

									<div>
										<label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Salary / comp</label>
										<input
											type="text"
											class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none"
											placeholder="e.g. £80,000"
											value={modalSalary()}
											onInput={(e) => setModalSalary(e.currentTarget.value)}
										/>
									</div>

									<div>
										<label class="mb-1 block text-xs font-medium text-[var(--sea-ink-soft)]">Notes</label>
										<textarea
											class="w-full rounded border border-[var(--line)] bg-[var(--surface)] px-3 py-2 text-sm text-[var(--sea-ink)] focus:outline-none resize-none"
											rows={3}
											placeholder="Any notes…"
											value={modalNotes()}
											onInput={(e) => setModalNotes(e.currentTarget.value)}
										/>
									</div>
								</div>

								<div class="mt-6 flex justify-end gap-3">
									<button
										type="button"
										onClick={closeModal}
										class="rounded-full border border-[var(--line)] px-4 py-1.5 text-sm text-[var(--sea-ink-soft)] transition hover:text-[var(--sea-ink)]"
									>
										Cancel
									</button>
									<button
										type="button"
										onClick={handleSubmit}
										disabled={createMutation.isPending || updateMutation.isPending}
										class="rounded-full border border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] px-4 py-1.5 text-sm font-semibold text-[var(--lagoon-deep)] transition hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)] disabled:opacity-50"
									>
										{modalMode() === "create" ? "Save" : "Update"}
									</button>
								</div>
							</div>
						</div>
					)
				}}
			</Show>
		</div>
	)
}
