import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createEffect, createSignal, For, on, Show } from "solid-js";
import { AddApplicationDialog } from "@/components/jobs/AddApplicationDialog";
import {
	TrackApplicationDialog,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { ToggleChip } from "@/components/ToggleChip";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	completeApps,
	needsCv,
	nextSelection,
	type Selection,
	selectedDraftApps,
} from "@/lib/applicationSelection";
import { formatChaseDate, isChaseOverdue } from "@/lib/chase";
import { CV_STAGE_LABEL, cvStage } from "@/lib/cvStage";
import { loadedData } from "@/lib/queryData";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import { cn } from "@/lib/utils";
import { StatusBadge } from "../../components/StatusBadge";
import {
	applicationStatusesQueryOptions,
	useApplicationStatuses,
} from "../../hooks/useApplicationStatuses";
import {
	applicationsQueryOptions,
	useApplications,
	useClearChase,
} from "../../hooks/useApplications";
import { queryClient } from "../../lib/queryClient";
import type { ApplicationWithDetails } from "../../types/application";
import type { Job } from "../../types/job";
import { CreateDraftsDialog } from "./-applications/CreateDraftsDialog";
import { DeleteApplicationDialog } from "./-applications/DeleteApplicationDialog";
import { SetAnotherChase } from "./-applications/SetAnotherChase";

function parseApplicationsSearch(search: Record<string, unknown>) {
	return {
		status: typeof search.status === "string" ? search.status : undefined,
		chase: search.chase === true || search.chase === "true" ? true : undefined,
	};
}

export const Route = createFileRoute("/_auth/applications")({
	validateSearch: parseApplicationsSearch,
	loader: ({ location }) => {
		const { status, chase } = parseApplicationsSearch(location.search);
		return Promise.all([
			queryClient.prefetchQuery(
				applicationsQueryOptions(status, chase === true),
			),
			queryClient.prefetchQuery(applicationStatusesQueryOptions),
		]);
	},
	component: ApplicationsPage,
});

function ApplicationsPage() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const query = useApplications(
		() => search().status,
		() => search().chase === true,
	);
	const clearChase = useClearChase();
	const statusesQuery = useApplicationStatuses();

	const [modalOpen, setModalOpen] = createSignal(false);
	const [editingApp, setEditingApp] =
		createSignal<ApplicationWithDetails | null>(null);
	const [chasedApps, setChasedApps] = createSignal<ApplicationWithDetails[]>(
		[],
	);
	const [deletingApp, setDeletingApp] =
		createSignal<ApplicationWithDetails | null>(null);

	const [addOpen, setAddOpen] = createSignal(false);
	const [addedJob, setAddedJob] = createSignal<
		Pick<Job, "ID" | "Title"> | undefined
	>();

	const [selection, setSelection] = createSignal<Selection>({});
	const [createOpen, setCreateOpen] = createSignal(false);
	let anchor: string | undefined;
	createEffect(
		on(
			() => search().status,
			() => {
				setSelection({});
				anchor = undefined;
			},
			{ defer: true },
		),
	);
	const toggleRow = (
		apps: ApplicationWithDetails[],
		id: string,
		extendRange: boolean,
	) => {
		setSelection(
			nextSelection(
				apps.map((a) => a.ID),
				selection(),
				anchor,
				id,
				extendRange,
			),
		);
		anchor = id;
	};
	const draftSelected = () =>
		selectedDraftApps(
			loadedData(query) ?? [],
			selection(),
			loadedData(statusesQuery) ?? [],
		);

	const startTracking = (job: Pick<Job, "ID" | "Title">) => {
		setEditingApp(null);
		setAddedJob(job);
		setAddOpen(false);
		setModalOpen(true);
	};

	const editing = () => {
		const app = editingApp();
		return app
			? {
					job: { ID: app.JobID, Title: app.JobTitle },
					existing: toExistingApp(app),
				}
			: undefined;
	};

	const openEdit = (app: ApplicationWithDetails) => {
		setAddedJob(undefined);
		setEditingApp(app);
		setModalOpen(true);
	};

	const markChased = (app: ApplicationWithDetails) =>
		clearChase.mutate(app.ID, {
			onSuccess: () => setChasedApps((prev) => [...prev, app]),
		});

	const isOverdue = (app: ApplicationWithDetails) =>
		app.ChaseBy !== null && isChaseOverdue(app.ChaseBy, new Date());

	const statusColour = (app: ApplicationWithDetails) => {
		if (app.StatusColour) return app.StatusColour;
		const status = statusesQuery.data?.find((s) => s.ID === app.StatusID);
		return status?.Colour ?? STATUS_FALLBACK_COLOUR;
	};

	return (
		<div class="px-7 py-6">
			<div class="flex items-start justify-between gap-4">
				<PageHeading
					title="Applications"
					subtitle="Track where each application stands"
				/>
				<Button type="button" onClick={() => setAddOpen(true)}>
					Add application
				</Button>
			</div>

			<div class="mb-4 flex flex-wrap items-center gap-1.5">
				<ToggleChip
					active={!search().status}
					onClick={() =>
						navigate({
							to: "/applications",
							search: { status: undefined, chase: search().chase },
						})
					}
				>
					All
				</ToggleChip>
				<For each={statusesQuery.data}>
					{(s) => (
						<ToggleChip
							active={search().status === s.ID}
							onClick={() =>
								navigate({
									to: "/applications",
									search: { status: s.ID, chase: search().chase },
								})
							}
						>
							<span
								class="inline-block size-2 rounded-full"
								style={{ background: s.Colour }}
							/>
							{s.Name}
						</ToggleChip>
					)}
				</For>
				<span class="mx-1 h-4 w-px bg-border" />
				<ToggleChip
					active={search().chase === true}
					onClick={() =>
						navigate({
							to: "/applications",
							search: {
								status: search().status,
								chase: search().chase ? undefined : true,
							},
						})
					}
				>
					Needs a chase
				</ToggleChip>
			</div>

			<For each={chasedApps()}>
				{(app) => (
					<SetAnotherChase
						applicationId={app.ID}
						title={app.JobTitle}
						onDone={() =>
							setChasedApps((prev) => prev.filter((p) => p.ID !== app.ID))
						}
					/>
				)}
			</For>

			<div class="flex items-start gap-6">
				<div class="min-w-0 flex-1">
					<QueryBoundary query={query} fallbackRows={5}>
						{(data) => (
							<Show
								when={data().length > 0}
								fallback={
									<Card class="p-10 text-center">
										<p class="text-sm text-muted">
											No applications yet. Track a job from the Jobs page.
										</p>
									</Card>
								}
							>
								<div class="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface">
									<For each={data()}>
										{(app) => (
											<div
												class={cn(
													"flex items-center gap-4 px-4 py-3 transition-colors hover:bg-surface-muted",
													search().chase &&
														isOverdue(app) &&
														"bg-destructive-subtle",
												)}
											>
												<input
													type="checkbox"
													aria-label={`Select ${app.JobTitle}`}
													class="size-4 shrink-0 cursor-pointer accent-primary"
													checked={!!selection()[app.ID]}
													onMouseDown={(e) => {
														if (e.shiftKey) e.preventDefault();
													}}
													onClick={(e) => {
														toggleRow(data(), app.ID, e.shiftKey);
														e.currentTarget.checked = !!selection()[app.ID];
													}}
												/>
												<div class="min-w-0 flex-1">
													<Link
														to="/applications/$id"
														params={{ id: app.ID }}
														class="block truncate text-sm font-medium text-foreground hover:underline"
													>
														{app.JobTitle}
													</Link>
													<p class="text-xs text-faint">
														{app.JobCompanySlug}
														{app.JobLocation ? ` · ${app.JobLocation}` : ""}
													</p>
												</div>
												<Show when={app.JobClosedAt}>
													<span class="shrink-0 rounded-full border border-border px-2 py-0.5 text-xs text-muted">
														Listing closed
													</span>
												</Show>
												<Show when={app.StatusName}>
													<StatusBadge
														name={app.StatusName}
														colour={statusColour(app)}
													/>
												</Show>
												<span
													class={cn(
														"shrink-0 text-xs",
														cvStage(app.Draft) === "failed"
															? "text-destructive-strong"
															: "text-muted",
													)}
													title={app.Draft?.LastError || undefined}
												>
													{CV_STAGE_LABEL[cvStage(app.Draft)]}
												</span>
												<Show when={app.AppliedAt}>
													<span class="shrink-0 font-mono text-xs tabular-nums text-faint">
														{app.AppliedAt}
													</span>
												</Show>
												<Show when={app.ChaseBy}>
													{(chaseBy) => (
														<span
															class={cn(
																"shrink-0 font-mono text-xs tabular-nums",
																isChaseOverdue(chaseBy(), new Date())
																	? "text-destructive-strong"
																	: "text-faint",
															)}
														>
															Chase {formatChaseDate(chaseBy())}
														</span>
													)}
												</Show>
												<Show when={app.ChaseBy}>
													<button
														type="button"
														disabled={clearChase.isPending}
														onClick={() => markChased(app)}
														class="shrink-0 rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
													>
														Chased
													</button>
												</Show>
												<button
													type="button"
													onClick={() => openEdit(app)}
													class="shrink-0 rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle"
												>
													Edit
												</button>
												<button
													type="button"
													onClick={() => setDeletingApp(app)}
													class="shrink-0 rounded px-2 py-1 text-xs font-medium text-destructive-strong transition hover:bg-destructive-subtle"
												>
													Delete
												</button>
											</div>
										)}
									</For>
								</div>
							</Show>
						)}
					</QueryBoundary>
				</div>
				<Show when={Object.keys(selection()).length > 0}>
					<aside class="sticky top-6 flex w-64 shrink-0 flex-col gap-3 rounded-xl border border-border bg-surface p-4">
						<div class="flex items-center justify-between">
							<span class="font-mono text-xs tabular-nums text-muted">
								{draftSelected().length} of {Object.keys(selection()).length}{" "}
								selected in Draft
							</span>
							<Button
								variant="ghost"
								size="sm"
								onClick={() => {
									setSelection({});
									anchor = undefined;
								}}
							>
								Clear
							</Button>
						</div>
						<Button
							disabled={draftSelected().filter(needsCv).length === 0}
							onClick={() => setCreateOpen(true)}
						>
							Create CV drafts ({draftSelected().filter(needsCv).length})
						</Button>
						<Button
							disabled={completeApps(draftSelected()).length === 0}
							onClick={() => {
								const run = completeApps(draftSelected()).map((a) => a.ID);
								const [first] = run;
								if (first)
									void navigate({
										to: "/applications/$id",
										params: { id: first },
										search: { run },
									});
							}}
						>
							Send complete ({completeApps(draftSelected()).length})
						</Button>
					</aside>
				</Show>
			</div>

			<CreateDraftsDialog
				open={createOpen()}
				onOpenChange={setCreateOpen}
				apps={draftSelected()}
			/>

			<TrackApplicationDialog
				open={modalOpen()}
				onOpenChange={setModalOpen}
				job={editing()?.job ?? addedJob()}
				existingApp={editing()?.existing}
			/>

			<AddApplicationDialog
				open={addOpen()}
				onOpenChange={setAddOpen}
				onAdded={startTracking}
			/>

			<DeleteApplicationDialog
				app={deletingApp()}
				onClose={() => setDeletingApp(null)}
			/>
		</div>
	);
}
