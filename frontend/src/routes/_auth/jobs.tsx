import { createFileRoute, Link } from "@tanstack/solid-router"
import { createSignal, For, Show } from "solid-js"
import { Badge } from "../../components/ui/badge"
import {
	Card,
	CardContent,
	CardFooter,
	CardHeader,
	CardTitle,
} from "../../components/ui/card"
import { queryClient } from "../../lib/queryClient"
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs"
import { useApplicationsForJobs, useCreateApplication, useUpdateApplication } from "../../hooks/useApplications"
import { useApplicationStatuses } from "../../hooks/useApplicationStatuses"

const PAGE_SIZE = 12

export const Route = createFileRoute("/_auth/jobs")({
	validateSearch: (search: Record<string, unknown>) => ({
		page: Math.max(1, Number(search.page) || 1),
	}),
	loader: () => queryClient.ensureQueryData(jobsQueryOptions),
	component: JobsPage,
})

function JobsPage() {
	const query = useJobs()
	const search = Route.useSearch()
	const statusesQuery = useApplicationStatuses()

	const jobs = () => query.data ?? []
	const page = () => search().page
	const totalPages = () => Math.ceil(jobs().length / PAGE_SIZE)
	const start = () => (page() - 1) * PAGE_SIZE
	const paginated = () => jobs().slice(start(), start() + PAGE_SIZE)
	const showingEnd = () => Math.min(start() + PAGE_SIZE, jobs().length)
	const pageJobIds = () => paginated().map((j) => j.ID)

	const appsForJobs = useApplicationsForJobs(pageJobIds)
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

	const pageNumbers = () => {
		const total = totalPages()
		const current = page()
		if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
		if (current <= 4) return [1, 2, 3, 4, 5, -1, total]
		if (current >= total - 3) return [1, -1, total - 4, total - 3, total - 2, total - 1, total]
		return [1, -1, current - 1, current, current + 1, -1, total]
	}

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
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					<For each={paginated()}>
						{(job, index) => {
							const appSummary = () => appsForJobs.data?.[job.ID]
							return (
								<div
									class="rise-in"
									style={{ "animation-delay": `${index() * 60}ms` }}
								>
									<Card class="h-full">
										<CardHeader>
											<div class="flex items-start justify-between gap-2">
												<CardTitle>{job.Title}</CardTitle>
												<Badge variant="secondary" class="shrink-0 uppercase tracking-wide">
													{job.Source}
												</Badge>
											</div>
										</CardHeader>
										<CardContent>
											<p class="text-sm font-medium text-[var(--sea-ink)]">
												{job.CompanySlug}
											</p>
											<p class="text-sm text-[var(--sea-ink-soft)]">{job.Location}</p>
										</CardContent>
										<CardFooter class="flex items-center gap-2 flex-wrap">
											<a
												href={job.URL}
												target="_blank"
												rel="noopener noreferrer"
												class="inline-flex items-center gap-1 rounded-full border border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] px-4 py-1.5 text-sm font-semibold text-[var(--lagoon-deep)] no-underline transition hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)]"
											>
												Apply →
											</a>
											<Show
												when={appSummary()}
												fallback={
													<button
														type="button"
														onClick={() => openTrack(job.ID)}
														class="inline-flex items-center rounded-full border border-[var(--line)] px-3 py-1.5 text-xs font-medium text-[var(--sea-ink-soft)] transition hover:border-[rgba(50,143,151,0.3)] hover:text-[var(--lagoon-deep)]"
													>
														Track
													</button>
												}
											>
												{(summary) => (
													<button
														type="button"
														onClick={() => openEdit(job.ID)}
														class="inline-flex items-center gap-1.5 rounded-full border border-[var(--line)] px-3 py-1.5 text-xs font-medium transition hover:border-[rgba(50,143,151,0.3)]"
													>
														<span
															class="h-2 w-2 rounded-full"
															style={{ background: summary().StatusColour || "#64748b" }}
														/>
														<span class="text-[var(--sea-ink)]">
															{summary().StatusName || "Tracked"}
														</span>
													</button>
												)}
											</Show>
										</CardFooter>
									</Card>
								</div>
							)
						}}
					</For>
				</div>

				<Show when={totalPages() > 1}>
					<div class="mt-10 flex flex-col items-center gap-4">
						<p class="text-xs text-[var(--sea-ink-soft)]">
							Showing {start() + 1}–{showingEnd()} of {jobs().length} jobs
						</p>
						<div class="flex items-center gap-1">
							<Link
								to="/jobs"
								search={{ page: page() - 1 }}
								disabled={page() === 1}
								aria-disabled={page() === 1}
								class={`inline-flex items-center rounded-full border px-3 py-1.5 text-sm font-medium no-underline transition ${
									page() === 1
										? "pointer-events-none border-[var(--line)] text-[var(--sea-ink-soft)] opacity-40"
										: "border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] text-[var(--lagoon-deep)] hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)]"
								}`}
							>
								← Prev
							</Link>

							<For each={pageNumbers()}>
								{(n) => (
									<Show
										when={n !== -1}
										fallback={
											<span class="px-1 text-sm text-[var(--sea-ink-soft)]">…</span>
										}
									>
										<Link
											to="/jobs"
											search={{ page: n }}
											class={`inline-flex h-8 w-8 items-center justify-center rounded-full border text-sm font-medium no-underline transition ${
												n === page()
													? "border-[rgba(50,143,151,0.5)] bg-[rgba(79,184,178,0.28)] text-[var(--sea-ink)]"
													: "border-transparent text-[var(--sea-ink-soft)] hover:border-[var(--line)] hover:text-[var(--sea-ink)]"
											}`}
										>
											{n}
										</Link>
									</Show>
								)}
							</For>

							<Link
								to="/jobs"
								search={{ page: page() + 1 }}
								disabled={page() === totalPages()}
								aria-disabled={page() === totalPages()}
								class={`inline-flex items-center rounded-full border px-3 py-1.5 text-sm font-medium no-underline transition ${
									page() === totalPages()
										? "pointer-events-none border-[var(--line)] text-[var(--sea-ink-soft)] opacity-40"
										: "border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] text-[var(--lagoon-deep)] hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)]"
								}`}
							>
								Next →
							</Link>
						</div>
					</div>
				</Show>
			</Show>

			{/* Track / Edit modal */}
			<Show when={trackingJobId()}>
				{(jobId) => {
					const job = () => paginated().find((j) => j.ID === jobId())
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
