import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createMemo, createSignal, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import {
	TrackApplicationDialog,
	toExistingApp,
} from "@/components/jobs/TrackApplicationDialog";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import type { JobFilters } from "@/lib/jobFilters";
import { applyJobFilters, parseSearch, sourceOptions } from "@/lib/jobFilters";
import { createJobColumns } from "../../components/jobs/columns";
import { JobsDataTable } from "../../components/jobs/JobsDataTable";
import { aiPrefsQueryOptions, useAiPrefs } from "../../hooks/useAiPrefs";
import { useApplications } from "../../hooks/useApplications";
import { useCompanies } from "../../hooks/useCompanies";
import { useAllJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";
import type { JobApplicationSummary } from "../../types/application";

export const Route = createFileRoute("/_auth/jobs")({
	validateSearch: (raw: Record<string, unknown>): Partial<JobFilters> =>
		parseSearch(raw),
	loader: () => queryClient.ensureQueryData(aiPrefsQueryOptions),
	component: JobsPage,
});

function JobsPage() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/jobs" });

	const query = useAllJobs();
	const aiPrefs = useAiPrefs();
	const jobs = () => query.data ?? [];
	const applications = useApplications();
	const appsForJobs = createMemo<Record<string, JobApplicationSummary>>(() =>
		Object.fromEntries(
			(applications.data ?? []).map((app) => [
				app.JobID,
				{
					ApplicationID: app.ID,
					StatusID: app.StatusID,
					StatusName: app.StatusName,
					StatusColour: app.StatusColour,
				},
			]),
		),
	);

	const companies = useCompanies();
	const companyName = () =>
		companies.data?.find((c) => c.ID === filters().company)?.Name ??
		"Selected company";

	const filters = () => parseSearch(search() as Record<string, unknown>);
	const filtered = createMemo(() => applyJobFilters(jobs(), filters()));
	const srcOptions = () => sourceOptions(jobs());

	const setFilters = (patch: Partial<JobFilters>) => {
		navigate({
			to: "/jobs",
			search: (p) => ({ ...p, ...patch }),
			replace: true,
		});
	};

	const [modalOpen, setModalOpen] = createSignal(false);
	const [trackingJobId, setTrackingJobId] = createSignal<string | null>(null);

	const openTrack = (jobId: string) => {
		setTrackingJobId(jobId);
		setModalOpen(true);
	};

	const openEdit = (jobId: string) => {
		if (!appsForJobs()[jobId]) return;
		setTrackingJobId(jobId);
		setModalOpen(true);
	};

	const currentJob = () => jobs().find((j) => j.ID === trackingJobId());
	const existingApp = () => {
		const jobId = trackingJobId();
		const app = applications.data?.find((a) => a.JobID === jobId);
		return app ? toExistingApp(app) : undefined;
	};

	const columns = createJobColumns({
		appsForJobs,
		onTrack: openTrack,
		onEdit: openEdit,
	});

	return (
		<div class="px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">Jobs</h1>
				<p class="mt-0.5 text-xs text-faint">
					Open roles scraped from your configured sources
				</p>
			</div>

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
							onClick={() => setFilters({ company: undefined })}
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
							onClick={() => setFilters({ scored: false })}
						>
							Scored for me
							<Icon name="x" size={12} />
						</Button>
					</Show>
				</div>
			</Show>

			<Show when={query.isSuccess}>
				<JobsDataTable
					columns={columns}
					data={filtered()}
					filters={filters()}
					onChange={setFilters}
					sourceOptions={srcOptions()}
				/>
			</Show>

			<TrackApplicationDialog
				open={modalOpen()}
				onOpenChange={setModalOpen}
				job={currentJob()}
				existingApp={existingApp()}
			/>
		</div>
	);
}
