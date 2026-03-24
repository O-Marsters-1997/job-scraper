import { createFileRoute } from "@tanstack/solid-router"
import { For } from "solid-js"
import { Badge } from "../components/ui/badge"
import {
	Card,
	CardContent,
	CardFooter,
	CardHeader,
	CardTitle,
} from "../components/ui/card"
import type { Job } from "../types/job"

export const Route = createFileRoute("/jobs")({
	loader: async (): Promise<Job[]> => {
		const res = await fetch("http://localhost:8080/jobs")
		if (!res.ok) throw new Error(`Failed to fetch jobs: ${res.status}`)
		return res.json() as Promise<Job[]>
	},
	pendingComponent: JobsPending,
	errorComponent: JobsError,
	component: JobsPage,
})

function JobsPage() {
	const jobs = Route.useLoaderData()

	return (
		<main class="page-wrap px-4 pb-12 pt-8">
			<div class="mb-8">
				<p class="island-kicker mb-2">Live Listings</p>
				<h1 class="display-title text-4xl font-bold text-[var(--sea-ink)] sm:text-5xl">
					Open Roles
				</h1>
			</div>
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				<For each={jobs()}>
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
		</main>
	)
}

function JobsPending() {
	return (
		<main class="page-wrap px-4 pb-12 pt-8">
			<p class="text-sm text-[var(--sea-ink-soft)]">Loading jobs…</p>
		</main>
	)
}

function JobsError({ error }: { error: Error }) {
	return (
		<main class="page-wrap px-4 pb-12 pt-8">
			<div class="island-shell rounded-xl p-6">
				<p class="island-kicker mb-2 text-red-600">Error</p>
				<p class="text-sm text-[var(--sea-ink-soft)]">{error.message}</p>
			</div>
		</main>
	)
}
