import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createMemo, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { TrackApplicationDialog } from "@/components/jobs/TrackApplicationDialog";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import type { JobFilters } from "@/lib/jobFilters";
import { applyJobFilters, parseSearch, sourceOptions } from "@/lib/jobFilters";
import { JobsDataTable } from "../../components/jobs/JobsDataTable";
import { aiPrefsQueryOptions, useAiPrefs } from "../../hooks/useAiPrefs";
import { useCompanies } from "../../hooks/useCompanies";
import { useAllJobs } from "../../hooks/useJobs";
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

	const query = useAllJobs();
	const aiPrefs = useAiPrefs();
	const jobs = () => query.data ?? [];
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

	const track = useTrackJobs(jobs);

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
					columns={track.columns}
					data={filtered()}
					filters={filters()}
					onChange={setFilters}
					sourceOptions={srcOptions()}
				/>
			</Show>

			<TrackApplicationDialog
				open={track.modalOpen()}
				onOpenChange={track.setModalOpen}
				job={track.currentJob()}
				existingApp={track.existingApp()}
			/>
		</div>
	);
}
