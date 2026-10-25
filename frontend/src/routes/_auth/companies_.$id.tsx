import { createFileRoute, Link } from "@tanstack/solid-router";
import { createEffect, createMemo, createSignal, on, Show } from "solid-js";
import { FavouriteStar } from "@/components/FavouriteStar";
import { GradeDialog } from "@/components/jobs/GradeDialog";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { agencyRollup, isAgencyKind } from "@/lib/agencyRollup";
import { formatDate } from "@/lib/datetime";
import {
	activeFilterCount,
	DEFAULT_FILTERS,
	type JobFilters,
} from "@/lib/jobFilters";
import { parseHistorySearch } from "@/lib/scrapeRuns";
import { outcomeLabels } from "@/types/recruiterContact";
import {
	companyQueryOptions,
	useCompany,
	useSetCompanyFavourite,
	useSetCompanyReview,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { announceBulkGrading, DismissToast } from "../../hooks/useDismissJob";
import { useJobPages } from "../../hooks/useJobs";
import { useRecruiterContactSummaries } from "../../hooks/useRecruiterContacts";
import { useSourceOptions } from "../../hooks/useSources";
import { useTrackJobs } from "../../hooks/useTrackJobs";
import { queryClient } from "../../lib/queryClient";
import { CompanyBoardsCard } from "./-companies-detail/CompanyBoardsCard";
import { CompanyDetailsCard } from "./-companies-detail/CompanyDetailsCard";
import { CompanyHistory } from "./-companies-detail/CompanyHistory";

const COMPANY_FILTERS: JobFilters = { ...DEFAULT_FILTERS, poster: "all" };

export const Route = createFileRoute("/_auth/companies_/$id")({
	loader: ({ params }) =>
		queryClient.prefetchQuery(companyQueryOptions(params.id)),
	validateSearch: parseHistorySearch,
	component: CompanyDetailPage,
});

function CompanyDetailPage() {
	const params = Route.useParams();
	const search = Route.useSearch();
	const navigate = Route.useNavigate();
	const companyQuery = useCompany(() => params().id);
	const trackMutation = useSetCompanyTracking();
	const reviewMutation = useSetCompanyReview();
	const favouriteMutation = useSetCompanyFavourite();

	const company = () =>
		companyQuery.isSuccess ? (companyQuery.data ?? undefined) : undefined;

	const [localFilters, setLocalFilters] = createSignal(COMPANY_FILTERS);
	createEffect(
		on(
			() => params().id,
			() => setLocalFilters(COMPANY_FILTERS),
			{ defer: true },
		),
	);
	const filters = () => ({
		...localFilters(),
		company: params().id,
	});
	const table = useJobPages(filters, (patch) =>
		setLocalFilters((f) => ({ ...f, ...patch })),
	);
	const sourceOptions = useSourceOptions();

	const hasFilters = () =>
		activeFilterCount(localFilters()) > 0 || localFilters().q !== "";

	const isAgency = () => isAgencyKind(company()?.Kind);
	const jobIds = createMemo(() => table.items().map((j) => j.ID));
	const summaries = useRecruiterContactSummaries(() =>
		isAgency() ? jobIds() : [],
	);
	const rollup = createMemo(() => agencyRollup(jobIds(), summaries.data ?? {}));

	const track = useTrackJobs(
		table.items,
		() => summaries.data,
		table.clearSelection,
	);

	return (
		<QueryBoundary
			query={companyQuery}
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
					<Show when={c().ReviewState === "dismissed"}>
						<Card class="mb-4">
							<CardContent class="flex items-center justify-between gap-4 pt-5">
								<p class="text-sm text-muted">
									You dismissed this company. It is no longer polled.
								</p>
								<Button
									variant="outline"
									size="sm"
									disabled={reviewMutation.isPending}
									onClick={() =>
										reviewMutation.mutate({ id: c().ID, state: "new" })
									}
								>
									Undo
								</Button>
							</CardContent>
						</Card>
					</Show>
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
							<Show when={isAgency()}>
								<dl class="mt-4 grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
									<div>
										<dt class="text-xs text-faint">Roles</dt>
										<dd class="font-semibold text-foreground">
											{rollup().roles}
										</dd>
									</div>
									<div>
										<dt class="text-xs text-faint">With a contact</dt>
										<dd class="font-semibold text-foreground">
											{rollup().contacted}
										</dd>
									</div>
									<div>
										<dt class="text-xs text-faint">Last contact</dt>
										<dd class="font-semibold text-foreground">
											<Show when={rollup().lastContactedOn} fallback="—">
												{(on) => formatDate(`${on()}T00:00:00`)}
											</Show>
										</dd>
									</div>
									<div>
										<dt class="text-xs text-faint">Best outcome</dt>
										<dd class="font-semibold text-foreground">
											<Show when={rollup().bestOutcome} fallback="—">
												{(outcome) => outcomeLabels[outcome()]}
											</Show>
										</dd>
									</div>
								</dl>
							</Show>
						</CardContent>
					</Card>

					<Tabs
						value={search().tab ?? "jobs"}
						onChange={(tab) =>
							navigate({
								search: (prev) => ({
									...prev,
									tab: tab === "history" ? "history" : undefined,
								}),
							})
						}
					>
						<TabsList variant="line" class="mb-3">
							<TabsTrigger value="jobs">Jobs</TabsTrigger>
							<TabsTrigger value="history">History</TabsTrigger>
						</TabsList>
						<TabsContent value="history">
							<CompanyHistory
								companyId={c().ID}
								intervalMinutes={c().CheckIntervalMinutes}
								range={search()}
								onRangeChange={(next) =>
									navigate({
										search: (prev) => ({
											tab: prev.tab,
											range: next.range,
											from: next.from,
											to: next.to,
										}),
									})
								}
							/>
						</TabsContent>
						<TabsContent value="jobs">
							<div class="grid grid-cols-1 items-start lg:grid-cols-[minmax(0,1fr)_284px] gap-4">
								<Card class="min-w-0">
									<CardHeader>
										<div class="flex items-center gap-2">
											<FavouriteStar
												favourite={c().Favourite ?? false}
												hourly={c().Tracked && c().CheckIntervalMinutes === 60}
												companyName={c().Name}
												disabled={favouriteMutation.isPending}
												onToggle={() =>
													favouriteMutation.mutate({
														id: c().ID,
														favourite: !c().Favourite,
													})
												}
											/>
											<CardTitle>Jobs at {c().Name}</CardTitle>
										</div>
									</CardHeader>
									<CardContent class="gap-0">
										<QueryBoundary query={table.query} fallbackRows={3}>
											{() => (
												<Show
													when={table.total() > 0 || hasFilters()}
													fallback={
														<p class="text-sm text-faint">
															No jobs from this company yet.
														</p>
													}
												>
													<JobsDataTable
														columns={
															isAgency() ? track.contactColumns : track.columns
														}
														data={table.items()}
														total={table.total()}
														page={table.page()}
														pageSize={table.pageSize}
														onPage={table.setPage}
														filters={filters()}
														onChange={table.setFilters}
														hidePosterFilter
														sourceOptions={sourceOptions()}
														selection={table.rowSelection()}
														onSelectionChange={table.select}
														selectedJobs={table.selectedItems()}
														bulkActions={[
															{
																label: "Grade",
																icon: "check",
																onSelect: track.openGrade,
															},
														]}
													/>
												</Show>
											)}
										</QueryBoundary>
									</CardContent>
								</Card>

								<div class="sticky top-0 flex flex-col gap-3">
									<CompanyBoardsCard companyId={params().id} />
									<CompanyDetailsCard company={c()} />
								</div>
							</div>
						</TabsContent>
					</Tabs>
					<DismissToast />
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
			)}
		</QueryBoundary>
	);
}
