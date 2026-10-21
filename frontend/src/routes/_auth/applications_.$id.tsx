import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createSignal, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { StatusBadge } from "@/components/StatusBadge";
import { CoverLetterCard } from "@/components/tailoring/CoverLetterCard";
import { JobDrafts } from "@/components/tailoring/JobDrafts";
import { CreateTaskDialog } from "@/components/tasks/CreateTaskDialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
	applicationsQueryOptions,
	useApplications,
} from "@/hooks/useApplications";
import { jobCommentsQueryOptions } from "@/hooks/useJobComments";
import { jobNoteQueryOptions } from "@/hooks/useJobNotes";
import { jobQueryOptions, useJob } from "@/hooks/useJobs";
import { jobDraftsQueryOptions } from "@/hooks/useTailoring";
import { jobOpenTasksQueryOptions } from "@/hooks/useTasks";
import { queryClient } from "@/lib/queryClient";
import { loadedData } from "@/lib/queryData";
import { CommentableDescription } from "./-jobs-detail/CommentableDescription";
import { JobNotesCard } from "./-jobs-detail/JobNotesCard";
import { JobTasksCard } from "./-jobs-detail/JobTasksCard";

export const Route = createFileRoute("/_auth/applications_/$id")({
	loader: async ({ params }) => {
		const apps = await queryClient.ensureQueryData(applicationsQueryOptions());
		const jobId = apps.find((a) => a.ID === params.id)?.JobID;
		if (!jobId) return;
		void queryClient.prefetchQuery(jobQueryOptions(jobId));
		void queryClient.prefetchQuery(jobDraftsQueryOptions(jobId));
		void queryClient.prefetchQuery(jobOpenTasksQueryOptions(jobId));
		void queryClient.prefetchQuery(jobNoteQueryOptions(jobId));
		void queryClient.prefetchQuery(jobCommentsQueryOptions(jobId));
	},
	component: ApplicationDetailPage,
});

function ApplicationDetailPage() {
	const params = Route.useParams();
	const query = useApplications();
	const navigate = useNavigate();
	const [tab, setTab] = createSignal("overview");
	const [taskOpen, setTaskOpen] = createSignal(false);
	const job = useJob(
		() => loadedData(query)?.find((a) => a.ID === params().id)?.JobID ?? "",
	);

	return (
		<QueryBoundary
			query={query}
			select={(apps) => apps.find((a) => a.ID === params().id)}
			notFound={
				<div class="flex h-[calc(100vh-14rem)] flex-col items-center justify-center gap-4 text-center">
					<Icon name="zoomIn" size={40} strokeWidth={1.5} class="text-faint" />
					<p class="text-base font-semibold text-foreground">
						Application not found
					</p>
					<Button as={Link} to="/applications" variant="outline" size="sm">
						← Back to Applications
					</Button>
				</div>
			}
		>
			{(app) => (
				<div class="px-4 py-6 md:px-7">
					<PageHeading
						title={app().JobTitle}
						subtitle={`${app().JobCompanySlug}${app().JobLocation ? ` · ${app().JobLocation}` : ""}`}
					>
						<div class="flex shrink-0 items-center gap-3">
							<StatusBadge
								name={app().StatusName}
								colour={app().StatusColour}
							/>
							<Link
								to="/jobs/$id"
								params={{ id: app().JobID }}
								class="text-sm font-medium text-primary hover:underline"
							>
								View job
							</Link>
							<Button
								size="sm"
								onClick={() =>
									navigate({
										to: "/jobs/$id/tailor",
										params: { id: app().JobID },
									})
								}
							>
								Tailor CV
							</Button>
						</div>
					</PageHeading>

					<Tabs value={tab()} onChange={setTab}>
						<TabsList variant="line" class="mb-3">
							<TabsTrigger value="overview">Overview</TabsTrigger>
							<TabsTrigger value="description">Description</TabsTrigger>
							<TabsTrigger value="notes">Notes</TabsTrigger>
						</TabsList>
						<TabsContent value="overview">
							<div class="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
								<div class="flex flex-col gap-3">
									<JobDrafts jobId={app().JobID} />
									<CoverLetterCard jobId={app().JobID} />
								</div>
								<JobTasksCard
									jobId={app().JobID}
									onAdd={() => setTaskOpen(true)}
								/>
							</div>
						</TabsContent>
						<TabsContent value="description">
							<Card>
								<CardHeader>
									<CardTitle>Description</CardTitle>
								</CardHeader>
								<CardContent class="gap-0">
									<Show
										when={loadedData(job)?.Description}
										fallback={
											<p class="text-sm text-faint">No description captured.</p>
										}
									>
										{(desc) => (
											<CommentableDescription
												html={desc()}
												jobId={app().JobID}
											/>
										)}
									</Show>
								</CardContent>
							</Card>
						</TabsContent>
						<TabsContent value="notes" forceMount>
							<JobNotesCard jobId={app().JobID} />
						</TabsContent>
					</Tabs>

					<CreateTaskDialog
						open={taskOpen()}
						onOpenChange={setTaskOpen}
						job={{ ID: app().JobID, Title: app().JobTitle }}
					/>
				</div>
			)}
		</QueryBoundary>
	);
}
