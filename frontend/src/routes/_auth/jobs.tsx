import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createMemo, createSignal, Show } from "solid-js";
import { TrackApplicationDialog } from "@/components/jobs/TrackApplicationDialog";
import { SkeletonList } from "@/components/ui/skeleton";
import type { JobFilters } from "@/lib/jobFilters";
import { applyJobFilters, parseSearch, sourceOptions } from "@/lib/jobFilters";
import { createJobColumns } from "../../components/jobs/columns";
import { JobsDataTable } from "../../components/jobs/JobsDataTable";
import { useApplicationsForJobs } from "../../hooks/useApplications";
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs";
import { aiPrefsQueryOptions, useAiPrefs } from "../../hooks/useAiPrefs";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/jobs")({
	// Return Partial so <Link to="/jobs"> callers don't need to supply search params.
	validateSearch: (raw: Record<string, unknown>): Partial<JobFilters> =>
		parseSearch(raw),
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(jobsQueryOptions),
			queryClient.ensureQueryData(aiPrefsQueryOptions),
		]),
	component: JobsPage,
});

function JobsPage() {
	const search = Route.useSearch();
	const navigate = useNavigate();

	const query = useJobs();
	const aiPrefs = useAiPrefs();
	const jobs = () => query.data ?? [];
	const allJobIds = () => jobs().map((j) => j.ID);

	const appsForJobs = useApplicationsForJobs(allJobIds);

	// Normalise partial URL params to a full JobFilters with defaults.
	const filters = () => parseSearch(search() as Record<string, unknown>);
	const filtered = createMemo(() => applyJobFilters(jobs(), filters()));
	const srcOptions = () => sourceOptions(jobs());

	const setFilters = (patch: Partial<JobFilters>) =>
		navigate({
			to: "/jobs",
			// Reset to page 1 whenever anything other than page itself changes.
			search: (p) => ({
				...p,
				...patch,
				...("page" in patch ? {} : { page: undefined }),
			}),
			replace: true,
		});

	const [modalOpen, setModalOpen] = createSignal(false);
	const [trackingJobId, setTrackingJobId] = createSignal<string | null>(null);

	const openTrack = (jobId: string) => {
		setTrackingJobId(jobId);
		setModalOpen(true);
	};

	const openEdit = (jobId: string) => {
		if (!appsForJobs.data?.[jobId]) return;
		setTrackingJobId(jobId);
		setModalOpen(true);
	};

	const currentJob = () => jobs().find((j) => j.ID === trackingJobId());
	const currentSummary = () => {
		const id = trackingJobId();
		return id ? appsForJobs.data?.[id] : undefined;
	};
	const existingApp = () => {
		const summary = currentSummary();
		if (!summary) return undefined;
		return { id: summary.ApplicationID, statusId: summary.StatusID };
	};

	const columns = createJobColumns({
		appsForJobs: () => appsForJobs.data,
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
					<Link to="/settings/ai" class="font-medium underline underline-offset-2">
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
