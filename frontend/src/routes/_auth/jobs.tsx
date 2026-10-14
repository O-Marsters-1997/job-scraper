import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { GradeDialog } from "@/components/jobs/GradeDialog";
import { TrackApplicationDialog } from "@/components/jobs/TrackApplicationDialog";
import { PageHeading } from "@/components/PageHeading";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import type { JobFilters } from "@/lib/jobFilters";
import { parseSearch } from "@/lib/jobFilters";
import { JobsDataTable } from "../../components/jobs/JobsDataTable";
import { aiPrefsQueryOptions, useAiPrefs } from "../../hooks/useAiPrefs";
import { useCompany } from "../../hooks/useCompanies";
import { announceBulkGrading, DismissToast } from "../../hooks/useDismissJob";
import { useJobPages } from "../../hooks/useJobs";
import { SeenToast } from "../../hooks/useSeenToast";
import { useSources } from "../../hooks/useSources";
import { useTrackJobs } from "../../hooks/useTrackJobs";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/jobs")({
	validateSearch: (raw: Record<string, unknown>): Partial<JobFilters> =>
		parseSearch(raw),
	loader: () => queryClient.ensureQueryData(aiPrefsQueryOptions),
	component: JobsPage,
});

function JobsPage() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/jobs" });

	const aiPrefs = useAiPrefs();

	const filters = () => parseSearch(search() as Record<string, unknown>);
	const table = useJobPages(filters, (patch) =>
		navigate({
			to: "/jobs",
			search: (p) => ({ ...p, ...patch }),
			replace: true,
		}),
	);
	const query = table.query;
	const company = useCompany(() => filters().company ?? "");
	const companyName = () => company.data?.Name ?? "Selected company";
	const sources = useSources();
	const sourceOptions = () =>
		(sources.data ?? []).map((s) => ({ value: s.name, label: s.label }));

	const track = useTrackJobs(
		table.items,
		() => undefined,
		table.clearSelection,
	);

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Jobs"
				subtitle="Open roles scraped from your configured sources"
			/>

			<Show when={aiPrefs.data && !aiPrefs.data.scoringEnabled}>
				<div class="mb-4 rounded-xl border border-accent-border bg-accent-subtle px-4 py-3 text-sm text-accent-text">
					AI scoring is off —{" "}
					<Link
						to="/settings/ai"
						class="font-medium underline underline-offset-2"
					>
						add a key in Settings
					</Link>
				</div>
			</Show>

			<Show when={query.isPending}>
				<SkeletonList rows={6} />
			</Show>

			<Show when={query.isError}>
				<div class="rounded-xl border border-destructive/30 bg-destructive-subtle p-6">
					<p class="mb-1 text-sm font-semibold text-destructive-strong">
						Error
					</p>
					<p class="text-sm text-muted">{query.error?.message}</p>
				</div>
			</Show>

			<Show when={filters().company || filters().scored}>
				<div class="mb-3 flex flex-wrap items-center gap-2">
					<Show when={filters().company}>
						<Button
							variant="outline"
							size="sm"
							aria-label={`Clear company filter: ${companyName()}`}
							onClick={() => table.setFilters({ company: undefined })}
						>
							{companyName()}
							<Icon name="x" size={12} />
						</Button>
					</Show>
					<Show when={filters().scored}>
						<Button
							variant="outline"
							size="sm"
							aria-label="Clear scored filter"
							onClick={() => table.setFilters({ scored: false })}
						>
							Scored for me
							<Icon name="x" size={12} />
						</Button>
					</Show>
				</div>
			</Show>

			<Show when={query.isSuccess}>
				<JobsDataTable
					columns={track.columns}
					data={table.items()}
					total={table.total()}
					page={table.page()}
					pageSize={table.pageSize}
					onPage={table.setPage}
					filters={filters()}
					onChange={table.setFilters}
					sourceOptions={sourceOptions()}
					sortControl
					selection={table.rowSelection()}
					onSelectionChange={table.select}
					selectedJobs={table.selectedItems()}
					onBulkGrade={track.openGrade}
					onBulkSeen={track.bulkSeen}
					onMarkAllSeen={track.markAllSeen}
				/>
			</Show>

			<DismissToast />
			<SeenToast />

			<GradeDialog
				open={track.gradeOpen()}
				onOpenChange={track.setGradeOpen}
				jobs={track.gradeJobs()}
				onSaved={(jobIds, priors) => {
					table.clearSelection();
					if (jobIds.length > 1) announceBulkGrading({ jobIds, priors });
				}}
			/>

			<TrackApplicationDialog
				open={track.modalOpen()}
				onOpenChange={track.setModalOpen}
				job={track.currentJob()}
				existingApp={track.existingApp()}
			/>
		</div>
	);
}
