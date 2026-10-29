import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import {
	createSignal,
	For,
	type JSX,
	onCleanup,
	onMount,
	Show,
} from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { StatusBadge } from "@/components/StatusBadge";
import { CoverLetterCard } from "@/components/tailoring/CoverLetterCard";
import { DownloadPdfButton } from "@/components/tailoring/DownloadPdfButton";
import { CreateTaskDialog } from "@/components/tasks/CreateTaskDialog";
import {
	Accordion,
	AccordionContent,
	AccordionItem,
	AccordionTrigger,
} from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
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
import { useCoverLetters } from "@/hooks/useCoverLetters";
import { useCVTemplates } from "@/hooks/useCVTemplates";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { jobCommentsQueryOptions } from "@/hooks/useJobComments";
import { jobNoteQueryOptions } from "@/hooks/useJobNotes";
import { jobQueryOptions, useJob } from "@/hooks/useJobs";
import {
	jobDraftsQueryOptions,
	useCreateDraft,
	useDiscardDraft,
	useJobDrafts,
} from "@/hooks/useTailoring";
import {
	jobOpenTasksQueryOptions,
	useJobOpenTasks,
	useSetTaskDone,
} from "@/hooks/useTasks";
import { checklistState, linkedDraft } from "@/lib/applicationChecklist";
import { runNeighbour } from "@/lib/applicationSelection";
import { formatChaseDate } from "@/lib/chase";
import { dayKey, formatDate } from "@/lib/datetime";
import { queryClient } from "@/lib/queryClient";
import { loadedData } from "@/lib/queryData";
import { cn, titleCase } from "@/lib/utils";
import type { ApplicationWithDetails } from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import type { CV } from "@/types/cv";
import type { Job } from "@/types/job";
import type { Draft } from "@/types/tailoring";
import { ChaseRow } from "./-jobs-detail/ChaseRow";
import { CommentableDescription } from "./-jobs-detail/CommentableDescription";
import { JobNotesCard } from "./-jobs-detail/JobNotesCard";
import { workStyle } from "./-jobs-detail/workStyle";

function parseApplicationSearch(search: Record<string, unknown>): {
	run?: string[];
} {
	const run = Array.isArray(search.run)
		? search.run.filter((id): id is string => typeof id === "string")
		: [];
	return run.length > 0 ? { run } : {};
}

export const Route = createFileRoute("/_auth/applications_/$id")({
	validateSearch: parseApplicationSearch,
	loader: async ({ params }) => {
		const apps = await queryClient.ensureQueryData(applicationsQueryOptions());
		const jobId = apps.find((a) => a.ID === params.id)?.JobID;
		if (!jobId) return;
		void queryClient.prefetchQuery(jobDraftsQueryOptions(jobId));
		void queryClient.prefetchQuery(jobOpenTasksQueryOptions(jobId));
		void queryClient.prefetchQuery(jobNoteQueryOptions(jobId));
		void queryClient.prefetchQuery(jobCommentsQueryOptions(jobId));
		await queryClient.prefetchQuery(jobQueryOptions(jobId));
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

function ChecklistRow(props: {
	value: string;
	done: boolean;
	inProgress?: boolean;
	title: string;
	summary: string;
	children: JSX.Element;
}) {
	return (
		<AccordionItem value={props.value}>
			<AccordionTrigger class="items-center gap-3 p-3 hover:no-underline">
				<span
					role="img"
					aria-label={
						props.done ? "Done" : props.inProgress ? "In progress" : "To do"
					}
					class={cn(
						"flex size-5 shrink-0 items-center justify-center rounded-full border",
						props.done
							? "border-primary bg-primary text-primary-foreground"
							: props.inProgress
								? "border-status-interview"
								: "border-border",
					)}
				>
					<Show when={props.done}>
						<Icon name="check" size={12} />
					</Show>
					<Show when={props.inProgress && !props.done}>
						<span class="size-2 rounded-full bg-status-interview" />
					</Show>
				</span>
				<span class={cn("min-w-0 flex-1 truncate", props.done && "text-muted")}>
					{props.title}
				</span>
				<span class="shrink-0 text-xs font-normal text-faint">
					{props.summary}
				</span>
			</AccordionTrigger>
			<AccordionContent class="flex flex-col gap-2 pr-0 pl-7">
				{props.children}
			</AccordionContent>
		</AccordionItem>
	);
}

function ApplyRow(props: {
	app: ApplicationWithDetails;
	company: string;
	done: boolean;
	onAsk: () => void;
	onApplied?: (() => void) | undefined;
}) {
	const statuses = useApplicationStatuses();
	const update = useUpdateApplication();
	const [asking, setAsking] = createSignal(false);
	const [opened, setOpened] = createSignal(false);
	const onFocus = () => {
		if (!opened()) return;
		setAsking(true);
		props.onAsk();
	};
	onMount(() => window.addEventListener("focus", onFocus));
	onCleanup(() => window.removeEventListener("focus", onFocus));

	const markApplied = () => {
		const applied = loadedData(statuses)?.find(
			(s) => s.SystemKey === "applied",
		);
		if (!applied) return;
		update.mutate(
			{
				id: props.app.ID,
				data: { status_id: applied.ID, applied_at: dayKey(new Date()) },
			},
			{
				onSuccess: () => {
					setAsking(false);
					props.onApplied?.();
				},
			},
		);
	};

	return (
		<ChecklistRow
			value="apply"
			done={props.done}
			title={`Apply on ${props.company}'s site`}
			summary={
				props.done
					? props.app.AppliedAt
						? `Applied ${formatDate(props.app.AppliedAt)}`
						: "Applied"
					: "Opens in a new tab"
			}
		>
			<Show
				when={!props.done}
				fallback={
					<p class="text-xs text-muted">
						Marked applied
						{props.app.AppliedAt && ` on ${formatDate(props.app.AppliedAt)}`}.
					</p>
				}
			>
				<Button
					as="a"
					href={props.app.JobURL}
					target="_blank"
					rel="noopener noreferrer"
					variant="outline"
					size="sm"
					class="w-fit"
					onClick={() => {
						setOpened(true);
						setAsking(true);
					}}
				>
					Open application
					<span aria-hidden="true">↗</span>
					<span class="sr-only">(opens in new tab)</span>
				</Button>
				<Show when={asking()}>
					<div class="flex flex-wrap items-center gap-2 text-sm">
						<span class="text-muted">Did you send it?</span>
						<Button size="sm" loading={update.isPending} onClick={markApplied}>
							Yes, I applied
						</Button>
						<Button
							variant="outline"
							size="sm"
							onClick={() => setAsking(false)}
						>
							Not yet
						</Button>
					</div>
				</Show>
			</Show>
		</ChecklistRow>
	);
}

function CvPicker(props: {
	cvs: CV[];
	value: CV | undefined;
	onChange: (cv: CV) => void;
}) {
	return (
		<Select
			options={props.cvs}
			optionValue={(c) => `${c?.DocID}/${c?.TabID}`}
			optionTextValue={(c) => c?.Title ?? ""}
			disallowEmptySelection
			value={props.value}
			onChange={(c) => c && props.onChange(c)}
			itemComponent={(p) => (
				<SelectItem item={p.item}>
					<SelectItemLabel>{p.item.rawValue?.Title}</SelectItemLabel>
				</SelectItem>
			)}
		>
			<SelectTrigger aria-label="Source CV" class="w-56">
				<Select.Value<CV>>{(s) => s.selectedOption()?.Title}</Select.Value>
			</SelectTrigger>
			<SelectContent />
		</Select>
	);
}

function CvRowBody(props: {
	jobId: string;
	draft: Draft | undefined;
	failed: Draft | undefined;
	done: boolean;
}) {
	const navigate = useNavigate();
	const cvs = useCVTemplates();
	const create = useCreateDraft();
	const discard = useDiscardDraft();
	const [picked, setPicked] = createSignal<CV>();
	const options = () => loadedData(cvs)?.filter((c) => c.Visible) ?? [];
	const chosen = () =>
		picked() ?? options().find((c) => c.Default) ?? options()[0];

	const openAsIs = useFormSubmit(async () => {
		const cv = chosen();
		if (!cv) return;
		const stale = props.draft?.outcome
			? undefined
			: (props.draft ?? props.failed);
		if (stale) await discard.mutateAsync(stale.id);
		const ref = await create.mutateAsync({
			jobId: props.jobId,
			docId: cv.DocID,
			tabId: cv.TabID,
			mode: "as_is",
		});
		await navigate({ to: "/tailoring/drafts/$id", params: { id: ref.id } });
	});
	const startOver = () => {
		if (window.confirm("Discard this draft and start a new one?"))
			void openAsIs.submit();
	};

	return (
		<Show
			when={props.draft}
			fallback={
				<>
					<p class="text-xs text-muted">
						{props.done
							? "Marked done because the application moved on. No tailored CV linked."
							: "No tailored CV for this role yet."}
					</p>
					<FormFeedback error={openAsIs.error()} />
					<div class="flex flex-wrap items-center gap-2">
						<CvPicker cvs={options()} value={chosen()} onChange={setPicked} />
						<Button
							size="sm"
							disabled={!chosen()}
							loading={openAsIs.pending()}
							onClick={() => void openAsIs.submit()}
						>
							Open as-is
						</Button>
					</div>
				</>
			}
		>
			{(draft) => (
				<>
					<Link
						to="/tailoring/drafts/$id"
						params={{ id: draft().id }}
						class="flex items-center justify-between gap-3 rounded-md py-1.5 pr-3 text-sm transition-colors hover:bg-surface-muted"
					>
						<span>
							Tailored CV, {formatDate(draft().createdAt)}
							<Show when={draft().insights}>
								{(i) => (
									<span class="ml-2 text-xs text-muted">
										Insights: {i().anchored}/{i().total} anchored
									</span>
								)}
							</Show>
						</span>
						<Badge
							variant={draft().outcome === "kept" ? "default" : "secondary"}
							class={cn(
								draft().outcome === "kept" && "bg-primary/10 text-primary",
							)}
						>
							{draft().outcome === "kept" ? "Complete" : "Draft"}
						</Badge>
					</Link>
					<Show when={draft().status === "ready" || draft().outcome === "kept"}>
						<div>
							<DownloadPdfButton
								draftId={draft().id}
								jobId={props.jobId}
								variant="outline"
							/>
						</div>
					</Show>
					<Show when={draft().outcome !== "kept"}>
						<FormFeedback error={openAsIs.error()} />
						<div class="flex flex-wrap items-center gap-2">
							<CvPicker cvs={options()} value={chosen()} onChange={setPicked} />
							<Button
								size="sm"
								onClick={() =>
									void navigate({
										to: "/tailoring/drafts/$id",
										params: { id: draft().id },
									})
								}
							>
								Open draft
							</Button>
							<Button
								variant="outline"
								size="sm"
								disabled={
									!chosen() ||
									(draft().status !== "ready" && draft().status !== "failed")
								}
								loading={openAsIs.pending()}
								onClick={startOver}
							>
								Start over
							</Button>
						</div>
					</Show>
				</>
			)}
		</Show>
	);
}

function ChecklistTab(props: {
	app: ApplicationWithDetails;
	onAddStep: () => void;
	onApplied?: () => void;
}) {
	const statuses = useApplicationStatuses();
	const drafts = useJobDrafts(() => props.app.JobID);
	const letters = useCoverLetters(() => props.app.JobID);
	const tasks = useJobOpenTasks(() => props.app.JobID);
	const setDone = useSetTaskDone();

	const [open, setOpen] = createSignal<string[]>([]);

	const openTasks = () => loadedData(tasks)?.items ?? [];
	const draft = () => linkedDraft(loadedData(drafts) ?? []);
	const state = () =>
		checklistState({
			statusSystemKey:
				loadedData(statuses)?.find((s) => s.ID === props.app.StatusID)
					?.SystemKey ?? "",
			hasKeptDraft:
				loadedData(drafts)?.some((d) => d.outcome === "kept") ?? false,
			hasOpenDraft: draft() !== undefined,
			hasLetter: (loadedData(letters)?.length ?? 0) > 0,
			hasDoneLetter: loadedData(letters)?.some((l) => l.doneAt) ?? false,
			applyTaskOpen: openTasks().some((t) => t.TypeSystemKey === "apply"),
			chaseBy: props.app.ChaseBy,
		});

	return (
		<Show when={loadedData(statuses) && loadedData(tasks)}>
			<div class="flex flex-col gap-3">
				<Accordion
					collapsible
					value={open()}
					onChange={setOpen}
					class="rounded-xl border border-border bg-surface"
				>
					<ChecklistRow
						value="cv"
						done={state().cv === "done"}
						inProgress={state().cv === "inProgress"}
						title="CV"
						summary={
							draft()?.outcome === "kept"
								? "Complete"
								: state().cv === "inProgress"
									? "Draft in progress"
									: draft()
										? "Draft"
										: state().cv === "done"
											? "No CV linked"
											: "Not started"
						}
					>
						<CvRowBody
							jobId={props.app.JobID}
							draft={draft()}
							failed={loadedData(drafts)?.find(
								(d) => !d.outcome && d.status === "failed",
							)}
							done={state().cv === "done"}
						/>
					</ChecklistRow>
					<ChecklistRow
						value="letter"
						done={state().coverLetter === "done"}
						inProgress={state().coverLetter === "inProgress"}
						title="Cover letter"
						summary={
							{
								done: "Done",
								inProgress: "Draft in progress",
								todo: "Optional",
							}[state().coverLetter]
						}
					>
						<p class="text-xs text-muted">
							Check whether the listing asks for a cover letter.
						</p>
						<details>
							<summary class="cursor-pointer text-xs text-muted">
								Write a cover letter
							</summary>
							<div class="mt-2">
								<CoverLetterCard jobId={props.app.JobID} />
							</div>
						</details>
					</ChecklistRow>
					<ApplyRow
						app={props.app}
						company={titleCase(props.app.JobCompanySlug)}
						done={state().apply === "done"}
						onAsk={() => setOpen(["apply"])}
						onApplied={props.onApplied}
					/>
					<ChecklistRow
						value="chase"
						done={state().followUp === "done"}
						title="Follow up"
						summary={
							props.app.ChaseBy
								? `Chase by ${formatChaseDate(props.app.ChaseBy)}`
								: "No chase set"
						}
					>
						<ChaseRow
							applicationId={props.app.ID}
							statusId={props.app.StatusID}
							chaseBy={props.app.ChaseBy}
						/>
					</ChecklistRow>
					<For each={openTasks().filter((t) => !t.TypeSystemKey)}>
						{(task) => (
							<ChecklistRow
								value={`task-${task.ID}`}
								done={false}
								title={task.TypeName}
								summary="Task"
							>
								<Show when={task.Notes}>
									<p class="text-xs text-muted">{task.Notes}</p>
								</Show>
								<Button
									variant="outline"
									size="sm"
									class="w-fit"
									loading={setDone.isPending}
									onClick={() => setDone.mutate({ id: task.ID, done: true })}
								>
									Done
								</Button>
							</ChecklistRow>
						)}
					</For>
				</Accordion>
				<Button
					variant="outline"
					size="sm"
					class="w-fit"
					onClick={props.onAddStep}
				>
					<Icon name="plus" size={14} />
					Add task
				</Button>
			</div>
		</Show>
	);
}

function RunBar(props: {
	run: string[];
	id: string;
	onGo: (id: string | undefined) => void;
}) {
	const back = () => runNeighbour(props.run, props.id, -1);
	const next = () => runNeighbour(props.run, props.id, 1);
	return (
		<div class="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-border bg-surface px-3 py-2 text-sm">
			<span class="font-mono text-xs tabular-nums text-muted">
				Send run {props.run.indexOf(props.id) + 1} of {props.run.length}
			</span>
			<div class="ml-auto flex items-center gap-1">
				<Button
					variant="ghost"
					size="sm"
					disabled={!back()}
					onClick={() => props.onGo(back())}
				>
					‹ Back
				</Button>
				<Button
					variant="ghost"
					size="sm"
					disabled={!next()}
					onClick={() => props.onGo(next())}
				>
					Skip ›
				</Button>
				<Button
					variant="outline"
					size="sm"
					onClick={() => props.onGo(undefined)}
				>
					End run
				</Button>
			</div>
		</div>
	);
}

function ApplicationDetailPage() {
	const params = Route.useParams();
	const search = Route.useSearch();
	const navigate = useNavigate();
	const goRun = (run: string[], id: string | undefined) =>
		void (id
			? navigate({ to: "/applications/$id", params: { id }, search: { run } })
			: navigate({
					to: "/applications",
					search: { status: undefined, chase: undefined },
				}));
	const query = useApplications();
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
					<Show when={search().run?.includes(app().ID) && search().run}>
						{(run) => (
							<RunBar
								run={run()}
								id={app().ID}
								onGo={(id) => goRun(run(), id)}
							/>
						)}
					</Show>
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
								<Show when={app().ID} keyed>
									<ChecklistTab
										app={app()}
										onAddStep={() => setTaskOpen(true)}
										onApplied={() => {
											const run = search().run;
											if (run?.includes(app().ID))
												goRun(run, runNeighbour(run, app().ID, 1));
										}}
									/>
								</Show>
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
						hideSystemTypes
					/>
				</div>
			)}
		</QueryBoundary>
	);
}
