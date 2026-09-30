import { createFileRoute, Link } from "@tanstack/solid-router";
import { createMemo, createSignal, Show } from "solid-js";
import { JobsDataTable } from "@/components/jobs/JobsDataTable";
import { TrackApplicationDialog } from "@/components/jobs/TrackApplicationDialog";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import {
	applyJobFilters,
	DEFAULT_FILTERS,
	filterCompanyJobs,
	sourceOptions,
} from "@/lib/jobFilters";
import {
	companiesQueryOptions,
	useCompanies,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { useAllJobs } from "../../hooks/useJobs";
import { useTrackJobs } from "../../hooks/useTrackJobs";
import { queryClient } from "../../lib/queryClient";
import { CompanyBoardsCard } from "./-companies-detail/CompanyBoardsCard";
import { CompanyDetailsCard } from "./-companies-detail/CompanyDetailsCard";

export const Route = createFileRoute("/_auth/companies_/$id")({
	loader: () => queryClient.prefetchQuery(companiesQueryOptions),
	component: CompanyDetailPage,
});

function CompanyDetailPage() {
	const params = Route.useParams();
	const companiesQuery = useCompanies();
	const jobsQuery = useAllJobs();
	const trackMutation = useSetCompanyTracking();

	const company = () => companiesQuery.data?.find((c) => c.ID === params().id);

	const [filters, setFilters] = createSignal(DEFAULT_FILTERS);
	const setFilterPatch = (patch: Partial<typeof DEFAULT_FILTERS>) =>
		setFilters((f) => ({ ...f, ...patch }));

	const jobsForCompany = createMemo(() => {
		const selected = company();
		if (!selected) return [];
		return filterCompanyJobs(jobsQuery.data ?? [], selected);
	});
	const companyJobs = createMemo(() =>
		applyJobFilters(jobsForCompany(), filters()),
	);

	const track = useTrackJobs(() => jobsQuery.data ?? []);

	return (
		<QueryBoundary
			query={companiesQuery}
			select={(list) => list.find((c) => c.ID === params().id)}
			notFound={
				<div class="flex h-[calc(100vh-14rem)] flex-col items-center justify-center gap-4 text-center">
					<div>
						<p class="text-base font-semibold text-foreground">
							Company not found
						</p>
					</div>
					<Button as={Link} to="/companies" variant="outline" size="sm">
						← Back to Companies
					</Button>
				</div>
			}
		>
			{(c) => (
				<div class="px-7 py-6 pb-16">
					<Card class="mb-4">
						<CardContent class="pt-5">
							<div class="flex items-start gap-4">
								<div class="flex h-[52px] w-[52px] shrink-0 items-center justify-center rounded-xl bg-accent-subtle text-base font-semibold text-accent-text">
									{c().Name.slice(0, 2).toUpperCase()}
								</div>
								<div class="min-w-0 flex-1">
									<h1 class="text-xl font-bold tracking-tight text-foreground">
										{c().Name}
									</h1>
									<div class="mt-2 flex flex-wrap items-center gap-2">
										<Show
											when={c().ATSSource}
											fallback={
												<span class="text-xs text-faint">discovery only</span>
											}
										>
											{(source) => <Badge variant="source">{source()}</Badge>}
										</Show>
									</div>
								</div>
								<Switch
									checked={c().Tracked}
									onChange={() =>
										trackMutation.mutate({
											id: c().ID,
											enabled: !c().Tracked,
										})
									}
									disabled={trackMutation.isPending}
								>
									<SwitchLabel class="flex shrink-0 items-center gap-2 text-xs text-faint">
										<SwitchControl>
											<SwitchThumb />
										</SwitchControl>
										Tracked
									</SwitchLabel>
								</Switch>
							</div>
						</CardContent>
					</Card>

					<div class="grid grid-cols-1 items-start lg:grid-cols-[1fr_284px] gap-4">
						<Card>
							<CardHeader>
								<CardTitle>Jobs at {c().Name}</CardTitle>
							</CardHeader>
							<CardContent class="gap-0">
								<Show when={jobsQuery.isPending}>
									<p class="text-sm text-muted">Loading jobs…</p>
								</Show>
								<Show when={jobsQuery.isError}>
									<p class="text-sm text-destructive-strong">
										Could not load jobs.
									</p>
								</Show>
								<Show
									when={jobsQuery.isSuccess && jobsForCompany().length > 0}
									fallback={
										<Show when={jobsQuery.isSuccess}>
											<p class="text-sm text-faint">
												No jobs from this company yet.
											</p>
										</Show>
									}
								>
									<JobsDataTable
										columns={track.columns}
										data={companyJobs()}
										filters={filters()}
										onChange={setFilterPatch}
										sourceOptions={sourceOptions(jobsForCompany())}
									/>
								</Show>
							</CardContent>
						</Card>

						<div class="sticky top-0 flex flex-col gap-3">
							<CompanyBoardsCard companyId={params().id} />
							<CompanyDetailsCard company={c()} />
						</div>
					</div>
					<TrackApplicationDialog
						open={track.modalOpen()}
						onOpenChange={track.setModalOpen}
						job={track.currentJob()}
						existingApp={track.existingApp()}
					/>
				</div>
			)}
		</QueryBoundary>
	);
}
