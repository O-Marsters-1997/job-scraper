import { createFileRoute, Link } from "@tanstack/solid-router";
import { For, type JSX, Show } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { dayKey, formatDate } from "@/lib/datetime";
import { cn, titleCase } from "@/lib/utils";
import { SourceBadge } from "../../components/SourceBadge";
import { StatusBadge } from "../../components/StatusBadge";
import {
	applicationStatusesQueryOptions,
	useApplicationStatuses,
} from "../../hooks/useApplicationStatuses";
import {
	applicationsQueryOptions,
	useApplications,
} from "../../hooks/useApplications";
import { allJobsQueryOptions, useAllJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/overview")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(allJobsQueryOptions),
			queryClient.ensureQueryData(applicationsQueryOptions()),
			queryClient.ensureQueryData(applicationStatusesQueryOptions),
		]),
	component: OverviewPage,
});

function isToday(iso: string): boolean {
	return dayKey(iso) === dayKey(new Date());
}

function OverviewPage() {
	const jobsQuery = useAllJobs();
	const appsQuery = useApplications();
	const statusesQuery = useApplicationStatuses();

	const jobs = () => jobsQuery.data ?? [];
	const applications = () => appsQuery.data ?? [];
	const statuses = () => statusesQuery.data ?? [];

	const totalJobs = () => jobs().length;
	const sourceCount = () => new Set(jobs().map((j) => j.Source)).size;

	const todayJobs = () => jobs().filter((j) => isToday(j.ScrapedAt));
	const newToday = () => todayJobs().length;
	const todaySources = () => new Set(todayJobs().map((j) => j.Source)).size;

	const totalApps = () => applications().length;
	const firstTwoIds = () =>
		statuses()
			.slice(0, 2)
			.map((s) => s.ID);
	const awaitingCount = () =>
		applications().filter((a) => firstTwoIds().includes(a.StatusID)).length;
	const respondedCount = () => totalApps() - awaitingCount();
	const responseRate = () =>
		totalApps() > 0 ? Math.round((respondedCount() / totalApps()) * 100) : 0;

	const appsByStatus = () => {
		const map: Record<string, number> = {};
		for (const app of applications()) {
			map[app.StatusID] = (map[app.StatusID] ?? 0) + 1;
		}
		return map;
	};

	const pipeline = () =>
		statuses().map((s) => ({ ...s, count: appsByStatus()[s.ID] ?? 0 }));

	const recentJobs = () =>
		[...jobs()]
			.sort(
				(a, b) =>
					new Date(b.ScrapedAt).getTime() - new Date(a.ScrapedAt).getTime(),
			)
			.slice(0, 5);
	const recentApps = () => applications().slice(0, 5);

	return (
		<div class="px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Overview
				</h1>
				<p class="mt-0.5 text-xs text-faint">Your job search at a glance</p>
			</div>

			<div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-[1.4fr_1.4fr_1.2fr]">
				<FeatureStat
					label="New today"
					value={newToday().toString()}
					delta={
						todaySources() > 0
							? `across ${todaySources()} ${todaySources() === 1 ? "source" : "sources"}`
							: "none scraped today"
					}
					cta={
						<Link
							to="/jobs"
							class="shrink-0 text-xs font-medium text-primary transition-colors hover:text-primary-hover"
						>
							Review jobs →
						</Link>
					}
				/>
				<FeatureStat
					label="Active applications"
					value={totalApps().toString()}
					delta={
						awaitingCount() > 0
							? `${awaitingCount()} awaiting response`
							: "all responded to"
					}
					deltaUp={awaitingCount() > 0}
					cta={
						<Link
							to="/applications"
							search={{ status: undefined }}
							class="shrink-0 text-xs font-medium text-primary transition-colors hover:text-primary-hover"
						>
							Manage →
						</Link>
					}
				/>
				<Card class="sm:col-span-2 lg:col-span-1">
					<div class="grid h-full grid-rows-2 divide-y divide-border">
						<MiniStat
							label="Total jobs"
							value={totalJobs().toString()}
							hint={`from ${sourceCount()} ${sourceCount() === 1 ? "source" : "sources"}`}
						/>
						<MiniStat
							label="Response rate"
							value={`${responseRate()}%`}
							hint={`${respondedCount()} of ${totalApps()} responded`}
						/>
					</div>
				</Card>
			</div>

			<div class="mb-4">
				<div class="mb-2 flex items-center justify-between">
					<span class="text-sm font-semibold text-foreground">
						Application pipeline
					</span>
					<Link
						to="/applications"
						search={{ status: undefined }}
						class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
					>
						View all →
					</Link>
				</div>
				<Show
					when={pipeline().length > 0}
					fallback={<p class="text-sm text-faint">No statuses configured.</p>}
				>
					<div class="flex gap-2">
						<For each={pipeline()}>
							{(seg) => (
								<Link
									to="/applications"
									search={{ status: seg.ID }}
									class="flex min-w-0 flex-1 items-center gap-2.5 rounded-xl border border-border bg-surface px-4 py-3 transition-colors hover:border-border-strong hover:bg-surface-muted"
								>
									<span
										class="size-2 shrink-0 rounded-full"
										style={{ "background-color": seg.Colour }}
									/>
									<span class="min-w-0 flex-1 truncate text-xs font-medium text-muted">
										{seg.Name}
									</span>
									<span class="font-mono text-sm font-medium tabular-nums text-foreground">
										{seg.count}
									</span>
								</Link>
							)}
						</For>
					</div>
				</Show>
			</div>

			<div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_300px]">
				<Card>
					<div class="flex items-center justify-between border-b border-border px-5 py-4">
						<h2 class="text-base font-semibold leading-snug text-foreground">
							Recent jobs
						</h2>
						<Link
							to="/jobs"
							class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
						>
							View all →
						</Link>
					</div>
					<Show
						when={recentJobs().length > 0}
						fallback={
							<p class="px-5 py-4 text-sm text-faint">No jobs scraped yet.</p>
						}
					>
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Role</TableHead>
									<TableHead>Company</TableHead>
									<TableHead>Source</TableHead>
									<TableHead>Scraped</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								<For each={recentJobs()}>
									{(job) => (
										<TableRow>
											<TableCell class="max-w-[220px] truncate font-medium text-foreground">
												{job.Title}
											</TableCell>
											<TableCell
												class="max-w-[160px] truncate text-muted"
												title={titleCase(job.CompanySlug)}
											>
												{titleCase(job.CompanySlug)}
											</TableCell>
											<TableCell>
												<SourceBadge source={job.Source} />
											</TableCell>
											<TableCell class="font-mono text-xs tabular-nums text-faint">
												{formatDate(job.ScrapedAt)}
											</TableCell>
										</TableRow>
									)}
								</For>
							</TableBody>
						</Table>
					</Show>
				</Card>

				<Card>
					<div class="flex items-center justify-between border-b border-border px-5 py-4">
						<h2 class="text-base font-semibold leading-snug text-foreground">
							Recent applications
						</h2>
						<Link
							to="/applications"
							search={{ status: undefined }}
							class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
						>
							View all →
						</Link>
					</div>
					<Show
						when={recentApps().length > 0}
						fallback={
							<p class="px-5 py-4 text-sm text-faint">No applications yet.</p>
						}
					>
						<For each={recentApps()}>
							{(app) => (
								<div class="flex items-center gap-3 border-b border-border px-4 py-3 last:border-0">
									<div class="min-w-0 flex-1">
										<p class="truncate text-sm font-medium text-foreground">
											{app.JobTitle}
										</p>
										<p class="text-xs text-faint">
											{titleCase(app.JobCompanySlug)}
											{app.JobLocation ? ` · ${app.JobLocation}` : ""}
										</p>
									</div>
									<Show when={app.StatusName}>
										<StatusBadge
											name={app.StatusName}
											colour={app.StatusColour}
										/>
									</Show>
								</div>
							)}
						</For>
					</Show>
				</Card>
			</div>
		</div>
	);
}

interface FeatureStatProps {
	label: string;
	value: string;
	delta: string;
	deltaUp?: boolean;
	cta?: JSX.Element;
}

function FeatureStat(props: FeatureStatProps) {
	return (
		<Card class="bg-accent-subtle/30 ring-1 ring-accent-border/40">
			<div class="flex h-full flex-col p-5">
				<div class="flex items-center justify-between gap-2">
					<p class="text-xs font-medium text-muted">{props.label}</p>
					{props.cta}
				</div>
				<p class="mt-3 font-mono text-4xl font-medium tabular-nums text-foreground">
					{props.value}
				</p>
				<p
					class={cn(
						"mt-auto pt-2 text-xs",
						props.deltaUp ? "text-primary" : "text-muted",
					)}
				>
					{props.delta}
				</p>
			</div>
		</Card>
	);
}

interface MiniStatProps {
	label: string;
	value: string;
	hint: string;
}

function MiniStat(props: MiniStatProps) {
	return (
		<div class="flex items-center justify-between gap-3 px-5 py-3.5">
			<div class="min-w-0">
				<p class="text-xs font-medium text-muted">{props.label}</p>
				<p class="mt-0.5 truncate text-xs text-faint">{props.hint}</p>
			</div>
			<p class="shrink-0 font-mono text-lg font-medium tabular-nums text-foreground">
				{props.value}
			</p>
		</div>
	);
}
