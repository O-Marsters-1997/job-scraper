import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import {
	TrackApplicationDialog,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Card } from "@/components/ui/card";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import { cn } from "@/lib/utils";
import { StatusBadge } from "../../components/StatusBadge";
import { useApplicationStatuses } from "../../hooks/useApplicationStatuses";
import { useApplications } from "../../hooks/useApplications";
import type { ApplicationWithDetails } from "../../types/application";
import { DeleteApplicationDialog } from "./-applications/DeleteApplicationDialog";

export const Route = createFileRoute("/_auth/applications")({
	validateSearch: (search: Record<string, unknown>) => ({
		status: typeof search.status === "string" ? search.status : undefined,
	}),
	component: ApplicationsPage,
});

function ApplicationsPage() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const query = useApplications(() => search().status);
	const statusesQuery = useApplicationStatuses();

	const [modalOpen, setModalOpen] = createSignal(false);
	const [editingApp, setEditingApp] =
		createSignal<ApplicationWithDetails | null>(null);
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

	const statusColour = (app: ApplicationWithDetails) => {
		if (app.StatusColour) return app.StatusColour;
		const status = statusesQuery.data?.find((s) => s.ID === app.StatusID);
		return status?.Colour ?? STATUS_FALLBACK_COLOUR;
	};

	const chipClass = (active: boolean) =>
		cn(
			"inline-flex cursor-pointer items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition",
			active
				? "border-primary bg-accent-subtle text-accent-text"
				: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
		);

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Applications"
				subtitle="Track where each application stands"
			/>

			<div class="mb-4 flex flex-wrap items-center gap-1.5">
				<button
					type="button"
					onClick={() =>
						navigate({ to: "/applications", search: { status: undefined } })
					}
					aria-pressed={!search().status}
					class={chipClass(!search().status)}
				>
					All
				</button>
				<For each={statusesQuery.data}>
					{(s) => (
						<button
							type="button"
							onClick={() =>
								navigate({ to: "/applications", search: { status: s.ID } })
							}
							aria-pressed={search().status === s.ID}
							class={chipClass(search().status === s.ID)}
						>
							<span
								class="inline-block size-2 rounded-full"
								style={{ background: s.Colour }}
							/>
							{s.Name}
						</button>
					)}
				</For>
			</div>

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
									<div class="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-surface-muted">
										<div class="min-w-0 flex-1">
											<p class="truncate text-sm font-medium text-foreground">
												{app.JobTitle}
											</p>
											<p class="text-xs text-faint">
												{app.JobCompanySlug}
												{app.JobLocation ? ` · ${app.JobLocation}` : ""}
											</p>
										</div>
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
