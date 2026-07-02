import { createFileRoute, Link } from "@tanstack/solid-router";
import { createMemo, createSignal, Show } from "solid-js";
import { createJobColumns } from "@/components/jobs/columns";
import { JobsDataTable } from "@/components/jobs/JobsDataTable";
import { SourceBadge } from "@/components/SourceBadge";
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
import { Switch, SwitchControl, SwitchThumb } from "@/components/ui/switch";
import { formatDate } from "@/lib/datetime";
import { applyJobFilters, DEFAULT_FILTERS } from "@/lib/jobFilters";
import {
	companiesQueryOptions,
	useCompanies,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs";
import { useUpdateSourceTarget } from "../../hooks/useSourceTargets";
import { queryClient } from "../../lib/queryClient";

const CHECK_INTERVAL_OPTIONS = [
	{ minutes: 60, label: "Hourly" },
	{ minutes: 180, label: "Every 3 hours" },
	{ minutes: 360, label: "Every 6 hours" },
	{ minutes: 720, label: "Every 12 hours" },
	{ minutes: 1440, label: "Daily" },
];

function FactRow(props: {
	label: string;
	children: import("solid-js").JSX.Element;
}) {
	return (
		<div class="flex items-start justify-between gap-4">
			<span class="shrink-0 text-xs text-faint">{props.label}</span>
			<span class="text-right text-xs text-foreground">{props.children}</span>
		</div>
	);
}

export const Route = createFileRoute("/_auth/companies_/$id")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(companiesQueryOptions),
			queryClient.ensureQueryData(jobsQueryOptions),
		]),
	component: CompanyDetailPage,
});

function CompanyDetailPage() {
	const params = Route.useParams();
	const companiesQuery = useCompanies();
	const jobsQuery = useJobs();
	const trackMutation = useSetCompanyTracking();
	const intervalMutation = useUpdateSourceTarget();

	const company = () => companiesQuery.data?.find((c) => c.ID === params().id);

	const [filters, setFilters] = createSignal(DEFAULT_FILTERS);
	const setFilterPatch = (patch: Partial<typeof DEFAULT_FILTERS>) =>
		setFilters((f) => ({ ...f, ...patch }));

	const jobsForCompany = createMemo(() => {
		const c = company();
		if (!c) return [];
		return (jobsQuery.data ?? []).filter((j) => j.CompanySlug === c.Slug);
	});
	const companyJobs = createMemo(() =>
		applyJobFilters(jobsForCompany(), filters()),
	);

	const columns = createJobColumns({
		appsForJobs: () => undefined,
		onTrack: () => {},
		onEdit: () => {},
	});

	return (
		<Show
			when={!companiesQuery.isPending}
			fallback={<div class="px-7 py-6 text-sm text-muted">Loading…</div>}
		>
			<Show
				when={company()}
				fallback={
					<div class="flex h-[calc(100vh-14rem)] flex-col items-center justify-center gap-4 text-center">
						<div>
							<p class="text-base font-semibold text-foreground">
								Company not found
							</p>
						</div>
						<Link to="/companies">
							<Button variant="outline" size="sm">
								← Back to Companies
							</Button>
						</Link>
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
									<div class="flex shrink-0 items-center gap-2">
										<span class="text-xs text-faint">Tracked</span>
										<Switch
											checked={c().Tracked}
											onChange={() =>
												trackMutation.mutate({
													id: c().ID,
													enabled: !c().Tracked,
												})
											}
											disabled={!c().ATSSource || trackMutation.isPending}
										>
											<SwitchControl>
												<SwitchThumb />
											</SwitchControl>
										</Switch>
									</div>
								</div>
							</CardContent>
						</Card>

						<div class="grid grid-cols-[1fr_284px] items-start gap-4">
							<Card>
								<CardHeader>
									<CardTitle>Jobs at {c().Name}</CardTitle>
								</CardHeader>
								<CardContent class="gap-0">
									<Show
										when={jobsForCompany().length > 0}
										fallback={
											<p class="text-sm text-faint">
												No jobs from this company yet.
											</p>
										}
									>
										<JobsDataTable
											columns={columns}
											data={companyJobs()}
											filters={filters()}
											onChange={setFilterPatch}
											sourceOptions={[]}
										/>
									</Show>
								</CardContent>
							</Card>

							<div class="sticky top-0 flex flex-col gap-3">
								<Card>
									<CardHeader class="pb-2">
										<CardTitle>Details</CardTitle>
									</CardHeader>
									<CardContent class="gap-2.5">
										<FactRow label="ATS">
											<Show when={c().ATSSource} fallback="—">
												{(source) => <SourceBadge source={source()} />}
											</Show>
										</FactRow>
										<Show when={c().ATSToken}>
											{(token) => (
												<FactRow label="Board token">
													<span class="font-mono text-xs">{token()}</span>
												</FactRow>
											)}
										</Show>
										<FactRow label="First seen">
											{formatDate(c().FirstSeenAt)}
										</FactRow>
										<FactRow label="Jobs">{c().JobCount}</FactRow>
										<Show when={c().Tracked && c().LastCheckedAt}>
											{(lastChecked) => (
												<FactRow label="Last checked">
													{formatDate(lastChecked())}
												</FactRow>
											)}
										</Show>
									</CardContent>
								</Card>

								<Show when={c().Tracked}>
									<Card>
										<CardHeader class="pb-2">
											<CardTitle>Check frequency</CardTitle>
										</CardHeader>
										<CardContent>
											<Select
												options={CHECK_INTERVAL_OPTIONS}
												optionValue="minutes"
												optionTextValue="label"
												value={
													CHECK_INTERVAL_OPTIONS.find(
														(o) => o.minutes === c().CheckIntervalMinutes,
													) ?? null
												}
												onChange={(opt) => {
													if (!opt || !c().TargetID) return;
													intervalMutation.mutate({
														id: c().TargetID,
														check_interval_minutes: opt.minutes,
													});
												}}
												itemComponent={(props) => (
													<SelectItem item={props.item}>
														<SelectItemLabel>
															{props.item.rawValue.label}
														</SelectItemLabel>
													</SelectItem>
												)}
											>
												<SelectTrigger>
													<Select.Value<
														(typeof CHECK_INTERVAL_OPTIONS)[number]
													>>
														{(state) =>
															state.selectedOption()?.label ?? "Select"
														}
													</Select.Value>
												</SelectTrigger>
												<SelectContent />
											</Select>
										</CardContent>
									</Card>
								</Show>
							</div>
						</div>
					</div>
				)}
			</Show>
		</Show>
	);
}
