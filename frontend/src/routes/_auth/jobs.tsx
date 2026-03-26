import { createFileRoute, Link } from "@tanstack/solid-router"
import { For, Show } from "solid-js"
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

	const jobs = () => query.data ?? []
	const page = () => search().page
	const totalPages = () => Math.ceil(jobs().length / PAGE_SIZE)
	const start = () => (page() - 1) * PAGE_SIZE
	const paginated = () => jobs().slice(start(), start() + PAGE_SIZE)
	const showingEnd = () => Math.min(start() + PAGE_SIZE, jobs().length)

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
						{(job, index) => (
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
									<CardFooter>
										<a
											href={job.URL}
											target="_blank"
											rel="noopener noreferrer"
											class="inline-flex items-center gap-1 rounded-full border border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] px-4 py-1.5 text-sm font-semibold text-[var(--lagoon-deep)] no-underline transition hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)]"
										>
											Apply →
										</a>
									</CardFooter>
								</Card>
							</div>
						)}
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
		</div>
	)
}
