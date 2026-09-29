import { createFileRoute, Link } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { DetailBoundary } from "@/components/DetailBoundary";
import { Icon } from "@/components/Icon";
import JobDescription from "@/components/jobs/JobDescription";
import { SuitabilityPanel } from "@/components/jobs/SuitabilityPanel";
import {
	TrackApplicationDialog,
	toApplicationSummary,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import { JobDrafts } from "@/components/tailoring/JobDrafts";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type {
	ApplicationWithDetails,
	JobApplicationSummary,
} from "@/types/application";
import {
	applicationsQueryOptions,
	useApplications,
} from "../../hooks/useApplications";
import { jobQueryOptions, useJob } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";
import { ApplicationCard } from "./-jobs-detail/ApplicationCard";
import { JobFactsCard } from "./-jobs-detail/JobFactsCard";
import { JobHeader } from "./-jobs-detail/JobHeader";

export const Route = createFileRoute("/_auth/jobs_/$id")({
	loader: ({ params }) =>
		Promise.all([
			queryClient.prefetchQuery(jobQueryOptions(params.id)),
			queryClient.prefetchQuery(applicationsQueryOptions()),
		]),
	component: JobDetailPage,
});

function JobDetailPage() {
	const params = Route.useParams();
	const jobsQuery = useJob(() => params().id);
	const appsQuery = useApplications();

	const app = (): ApplicationWithDetails | undefined =>
		appsQuery.data?.find((a) => a.JobID === params().id);
	const appSummary = (): JobApplicationSummary | undefined => {
		const a = app();
		return a && toApplicationSummary(a);
	};

	const [modalOpen, setModalOpen] = createSignal(false);

	const openTrack = () => setModalOpen(true);
	const openEdit = () => {
		if (!app()) return;
		setModalOpen(true);
	};

	const existingApp = () => {
		const a = app();
		return a ? toExistingApp(a) : undefined;
	};

	return (
		<DetailBoundary
			query={jobsQuery}
			data={jobsQuery.data}
			notFound={
				<div class="flex h-[calc(100vh-14rem)] flex-col items-center justify-center gap-4 text-center">
					<Icon name="zoomIn" size={40} strokeWidth={1.5} class="text-faint" />
					<div>
						<p class="text-base font-semibold text-foreground">Job not found</p>
						<p class="mt-1 text-sm text-muted">
							No job with ID{" "}
							<code class="rounded bg-surface-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
								{params().id}
							</code>
						</p>
					</div>
					<Button as={Link} to="/jobs" variant="outline" size="sm">
						← Back to Jobs
					</Button>
				</div>
			}
		>
			{(j) => (
				<div class="px-7 py-6 pb-16">
					<JobHeader
						job={j()}
						app={app()}
						appSummary={appSummary()}
						onTrack={openTrack}
						onEdit={openEdit}
					/>

					<div class="grid grid-cols-1 items-start lg:grid-cols-[1fr_284px] gap-4">
						<Card>
							<CardHeader>
								<CardTitle>Description</CardTitle>
							</CardHeader>
							<CardContent class="gap-0">
								<Show
									when={j().Description}
									fallback={
										<p class="text-sm text-faint">
											No description captured.{" "}
											<a
												href={j().URL}
												target="_blank"
												rel="noopener noreferrer"
												class="text-primary transition-colors hover:underline"
											>
												View the original listing ↗
											</a>
										</p>
									}
								>
									{(desc) => <JobDescription html={desc()} />}
								</Show>
							</CardContent>
						</Card>

						<div class="sticky top-0 flex flex-col gap-3">
							<JobFactsCard job={j()} />

							<Show when={(j().Skills?.length ?? 0) > 0}>
								<Card>
									<CardHeader class="pb-2">
										<CardTitle>Skills & technologies</CardTitle>
									</CardHeader>
									<CardContent>
										<div class="flex flex-wrap gap-1.5">
											<For each={j().Skills}>
												{(skill) => (
													<span class="rounded-full border border-border bg-surface px-2.5 py-0.5 text-xs text-muted">
														{skill}
													</span>
												)}
											</For>
										</div>
									</CardContent>
								</Card>
							</Show>

							<SuitabilityPanel job={j()} />

							<JobDrafts jobId={params().id} />

							<ApplicationCard
								app={app()}
								onTrack={openTrack}
								onEdit={openEdit}
							/>
						</div>
					</div>

					<TrackApplicationDialog
						open={modalOpen()}
						onOpenChange={setModalOpen}
						job={j()}
						existingApp={existingApp()}
					/>
				</div>
			)}
		</DetailBoundary>
	);
}
