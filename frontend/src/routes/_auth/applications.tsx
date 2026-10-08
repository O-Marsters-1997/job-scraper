import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import {
	TrackApplicationDialog,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { ToggleChip } from "@/components/ToggleChip";
import { Card } from "@/components/ui/card";
import { formatChaseDate, isChaseOverdue } from "@/lib/chase";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import { cn } from "@/lib/utils";
import { StatusBadge } from "../../components/StatusBadge";
import { useApplicationStatuses } from "../../hooks/useApplicationStatuses";
import { useApplications, useClearChase } from "../../hooks/useApplications";
import type { ApplicationWithDetails } from "../../types/application";
import { DeleteApplicationDialog } from "./-applications/DeleteApplicationDialog";
import { SetAnotherChase } from "./-applications/SetAnotherChase";

export const Route = createFileRoute("/_auth/applications")({
	validateSearch: (search: Record<string, unknown>) => ({
		status: typeof search.status === "string" ? search.status : undefined,
		chase: search.chase === true || search.chase === "true" ? true : undefined,
	}),
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
			<PageHeading
				title="Applications"
				subtitle="Track where each application stands"
			/>

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
										<div class="min-w-0 flex-1">
											<p class="truncate text-sm font-medium text-foreground">
												{app.JobTitle}
											</p>
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

			<TrackApplicationDialog
				open={modalOpen()}
				onOpenChange={setModalOpen}
				job={editing()?.job}
				existingApp={editing()?.existing}
			/>

			<DeleteApplicationDialog
				app={deletingApp()}
				onClose={() => setDeletingApp(null)}
			/>
		</div>
	);
}
