import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { StatusBadge } from "@/components/StatusBadge";
import { CoverLetterCard } from "@/components/tailoring/CoverLetterCard";
import { JobDrafts } from "@/components/tailoring/JobDrafts";
import { CreateTaskDialog } from "@/components/tasks/CreateTaskDialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApplicationStatuses } from "@/hooks/useApplicationStatuses";
import {
	applicationsQueryOptions,
	useApplications,
	useUpdateApplication,
} from "@/hooks/useApplications";
import { jobCommentsQueryOptions } from "@/hooks/useJobComments";
import { jobNoteQueryOptions } from "@/hooks/useJobNotes";
import { jobQueryOptions, useJob } from "@/hooks/useJobs";
import { jobDraftsQueryOptions } from "@/hooks/useTailoring";
import { jobOpenTasksQueryOptions } from "@/hooks/useTasks";
import { dayKey, formatDate } from "@/lib/datetime";
import { queryClient } from "@/lib/queryClient";
import { loadedData } from "@/lib/queryData";
import { titleCase } from "@/lib/utils";
import type { ApplicationWithDetails } from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Job } from "@/types/job";
import { CommentableDescription } from "./-jobs-detail/CommentableDescription";
import { JobNotesCard } from "./-jobs-detail/JobNotesCard";
import { JobTasksCard } from "./-jobs-detail/JobTasksCard";
import { workStyle } from "./-jobs-detail/workStyle";

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

function facts(app: ApplicationWithDetails, job: Job | undefined) {
	return [
		["Salary", job?.SalaryRange || app.SalaryInfo],
		["Work style", workStyle(job?.DaysInOffice)],
		["Type", job?.EmploymentType],
		["Tracked", formatDate(app.CreatedAt)],
	].filter(([, v]) => v);
}

function StatusSelect(props: { app: ApplicationWithDetails }) {
	const statuses = useApplicationStatuses();
	const update = useUpdateApplication();
	const options = () => loadedData(statuses) ?? [];
	return (
		<Select
			options={options()}
			optionValue={(s) => s?.ID ?? ""}
			optionTextValue={(s) => s?.Name ?? ""}
			disallowEmptySelection
			disabled={update.isPending}
			value={options().find((s) => s.ID === props.app.StatusID)}
			onChange={(s) => {
				if (!s || s.ID === props.app.StatusID) return;
				update.mutate({
					id: props.app.ID,
					data: {
						status_id: s.ID,
						...(s.SystemKey === "applied" && {
							applied_at: dayKey(new Date()),
						}),
					},
				});
			}}
			itemComponent={(p) => (
				<SelectItem item={p.item}>
					<SelectItemLabel>{p.item.rawValue?.Name}</SelectItemLabel>
				</SelectItem>
			)}
		>
			<SelectTrigger aria-label="Status" class="w-44">
				<Select.Value<ApplicationStatus>>
					{(state) => {
						const s = state.selectedOption();
						return s && <StatusBadge name={s.Name} colour={s.Colour} />;
					}}
				</Select.Value>
			</SelectTrigger>
			<SelectContent />
		</Select>
	);
}

function ApplicationDetailPage() {
	const params = Route.useParams();
	const query = useApplications();
	const navigate = useNavigate();
	const [tab, setTab] = createSignal("checklist");
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
					<div class="mb-4 flex flex-wrap items-start justify-between gap-3">
						<div class="min-w-0">
							<h1 class="text-lg font-bold tracking-tight text-foreground">
								{app().JobTitle}
							</h1>
							<p class="mt-0.5 text-sm text-muted">
								{titleCase(app().JobCompanySlug)}
								{app().JobLocation ? ` · ${app().JobLocation}` : ""}
							</p>
						</div>
						<div class="flex shrink-0 items-center gap-3">
							<StatusSelect app={app()} />
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
					</div>

					<div class="grid grid-cols-1 gap-4 md:grid-cols-[1fr_284px] md:gap-6">
						<aside class="flex flex-col gap-1 text-xs md:order-last md:gap-5">
							<div class="flex flex-wrap items-center gap-x-3 gap-y-1 md:flex-col md:items-start md:gap-2">
								<a
									href={app().JobURL}
									target="_blank"
									rel="noopener noreferrer"
									class="flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
								>
									Original listing
									<Icon name="externalLink" size={13} />
								</a>
								<Link
									to="/jobs/$id"
									params={{ id: app().JobID }}
									class="text-sm text-muted hover:text-foreground hover:underline"
								>
									Job details & score
								</Link>
							</div>
							<dl class="flex flex-wrap gap-x-3 gap-y-1 md:grid md:gap-2">
								<For each={facts(app(), loadedData(job))}>
									{([k, v]) => (
										<div class="flex gap-1.5 md:justify-between md:gap-3">
											<dt class="text-faint">{k}</dt>
											<dd class="text-right text-foreground">{v}</dd>
										</div>
									)}
								</For>
							</dl>
						</aside>

						<Tabs value={tab()} onChange={setTab} class="min-w-0">
							<TabsList variant="line" class="mb-3">
								<TabsTrigger value="checklist">Checklist</TabsTrigger>
								<TabsTrigger value="description">Description</TabsTrigger>
								<TabsTrigger value="notes">Notes</TabsTrigger>
							</TabsList>
							<TabsContent value="checklist">
								<div class="flex flex-col gap-4">
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
												<p class="text-sm text-faint">
													No description captured.
												</p>
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
							<TabsContent value="notes">
								<JobNotesCard jobId={app().JobID} />
							</TabsContent>
						</Tabs>
					</div>

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
