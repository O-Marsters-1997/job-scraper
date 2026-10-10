import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createEffect, createSignal, For, Show } from "solid-js";
import { recordEvent } from "@/api/events";
import { Icon } from "@/components/Icon";
import { CvSheet } from "@/components/jobs/CvSheet";
import { JobActionBar } from "@/components/jobs/JobActionBar";
import JobDescription from "@/components/jobs/JobDescription";
import { SuitabilityPanel } from "@/components/jobs/SuitabilityPanel";
import { TrackApplicationDialog } from "@/components/jobs/TrackApplicationDialog";
import { QueryBoundary } from "@/components/QueryBoundary";
import { JobDrafts } from "@/components/tailoring/JobDrafts";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Job } from "@/types/job";
import { applicationsQueryOptions } from "../../hooks/useApplications";
import { DismissToast, useDismissJob } from "../../hooks/useDismissJob";
import { jobQueryOptions, useJob, useMarkJobsSeen } from "../../hooks/useJobs";
import { useTrackJobs } from "../../hooks/useTrackJobs";
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
	const [cvOpen, setCvOpen] = createSignal(false);
	const jobsQuery = useJob(() => params().id);
	const navigate = useNavigate();
	const { dismiss } = useDismissJob();
	const dismissAndLeave = async (job: Job) => {
		await dismiss(job);
		navigate({ to: "/jobs" });
	};
	const markSeen = useMarkJobsSeen();
	let markedId: string | undefined;
	createEffect(() => {
		const job = jobsQuery.data;
		if (!job || job.Seen || markedId === job.ID) return;
		markedId = job.ID;
		markSeen.mutate({ jobIds: [job.ID], seen: true });
	});
	let openedId: string | undefined;
	createEffect(() => {
		const job = jobsQuery.data;
		if (!job || openedId === job.ID) return;
		openedId = job.ID;
		recordEvent({ type: "job_opened", subject_id: job.ID });
		if (new URLSearchParams(window.location.search).get("from") === "alert") {
			recordEvent({ type: "alert_opened", subject_id: job.ID });
		}
	});
	const track = useTrackJobs(() => (jobsQuery.data ? [jobsQuery.data] : []));

	const app = () =>
		track.applications.data?.find((a) => a.JobID === params().id);

	return (
		<QueryBoundary
			query={jobsQuery}
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
				<div class="px-4 py-6 pb-24 md:px-7 md:pb-16">
					<JobHeader
						job={j()}
						app={app()}
						appSummary={track.appsForJobs()[params().id]}
						onTrack={() => track.openTrack(params().id)}
					/>

					<div class="grid grid-cols-1 items-start lg:grid-cols-[1fr_284px] gap-4">
						<Card class="max-md:order-2">
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

						<div class="sticky top-0 flex flex-col gap-3 max-md:contents">
							<div class="max-md:order-3">
								<JobFactsCard job={j()} />
							</div>

							<Show when={(j().Skills?.length ?? 0) > 0}>
								<Card class="max-md:order-3">
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

							<div class="max-md:order-1">
								<SuitabilityPanel job={j()} />
							</div>

							<div id="job-drafts" class="max-md:order-4">
								<JobDrafts jobId={params().id} />
							</div>

							<div class="max-md:order-4">
								<ApplicationCard
									app={app()}
									onTrack={() => track.openTrack(params().id)}
								/>
							</div>
						</div>
					</div>

					<CvSheet job={j()} open={cvOpen()} onOpenChange={setCvOpen} />
					<JobActionBar
						job={j()}
						app={app()}
						onCv={() => setCvOpen(true)}
						onTrack={() => track.openTrack(params().id)}
						onDismiss={() => void dismissAndLeave(j())}
					/>
					<DismissToast />

					<TrackApplicationDialog
						open={track.modalOpen()}
						onOpenChange={track.setModalOpen}
						job={track.currentJob()}
						existingApp={track.existingApp()}
					/>
				</div>
			)}
		</QueryBoundary>
	);
}
