import { createFileRoute, Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { Card } from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { formatDate } from "@/lib/datetime";
import { cn } from "@/lib/utils";
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
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/overview")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(jobsQueryOptions),
			queryClient.ensureQueryData(applicationsQueryOptions()),
			queryClient.ensureQueryData(applicationStatusesQueryOptions),
		]),
	component: OverviewPage,
});

function titleCase(slug: string): string {
	return slug
		.split(/[-_\s]+/)
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.join(" ");
}

function isToday(iso: string): boolean {
	const d = new Date(iso);
	const now = new Date();
	return (
		d.getFullYear() === now.getFullYear() &&
		d.getMonth() === now.getMonth() &&
		d.getDate() === now.getDate()
	);
}

function OverviewPage() {
	const jobsQuery = useJobs();
	const appsQuery = useApplications();
	const statusesQuery = useApplicationStatuses();

	const jobs = () => jobsQuery.data ?? [];
	const applications = () => appsQuery.data ?? [];
	const statuses = () => statusesQuery.data ?? [];

	// ── Stat derivations ──────────────────────────────────────────────────────
	const totalJobs = () => jobs().length;
	const sourceCount = () => new Set(jobs().map((j) => j.Source)).size;

	const todayJobs = () => jobs().filter((j) => isToday(j.ScrapedAt));
	const newToday = () => todayJobs().length;
	const todaySources = () => new Set(todayJobs().map((j) => j.Source)).size;

	const totalApps = () => applications().length;
	// First two statuses (Saved + Applied) = still awaiting a response
	const firstTwoIds = () =>
		statuses()
			.slice(0, 2)
			.map((s) => s.ID);
	const awaitingCount = () =>
		applications().filter((a) => firstTwoIds().includes(a.StatusID)).length;
	const respondedCount = () => totalApps() - awaitingCount();
	const responseRate = () =>
		totalApps() > 0 ? Math.round((respondedCount() / totalApps()) * 100) : 0;

	// ── Pipeline counts ───────────────────────────────────────────────────────
	const appsByStatus = () => {
		const map: Record<string, number> = {};
		for (const app of applications()) {
			map[app.StatusID] = (map[app.StatusID] ?? 0) + 1;
		}
		return map;
	};

	const pipeline = () =>
		statuses().map((s) => ({ ...s, count: appsByStatus()[s.ID] ?? 0 }));

	// ── Recents ───────────────────────────────────────────────────────────────
	const recentJobs = () => jobs().slice(0, 5);
	const recentApps = () => applications().slice(0, 5);

	return (
		<div class="px-7 py-6">
			{/* Page header */}
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Overview
				</h1>
				<p class="mt-0.5 text-xs text-faint">Your job search at a glance</p>
			</div>

			{/* Stat cards */}
			<div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
				<StatCard
					label="Jobs in database"
					value={totalJobs().toString()}
					delta={`from ${sourceCount()} sources`}
				/>
				<StatCard
					label="New today"
					value={newToday().toString()}
					delta={
						todaySources() > 0
							? `across ${todaySources()} ${todaySources() === 1 ? "source" : "sources"}`
							: "none scraped today"
					}
				/>
				<StatCard
					label="Active applications"
					value={totalApps().toString()}
					delta={
						awaitingCount() > 0
							? `${awaitingCount()} awaiting response`
							: "all responded to"
					}
					deltaUp={awaitingCount() > 0}
				/>
				<StatCard
					label="Response rate"
					value={`${responseRate()}%`}
					delta={`${respondedCount()} of ${totalApps()} responded`}
				/>
			</div>

			{/* Application pipeline */}
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

			{/* Dashboard grid: recent jobs (wide) + recent applications (fixed) */}
			<div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_300px]">
				{/* Recent jobs */}
				<Card>
					<div class="flex items-center justify-between border-b border-border px-5 py-4">
						<h3 class="text-base font-semibold leading-snug text-foreground">
							Recent jobs
						</h3>
						<Link
							to="/jobs"
							class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
						>
							View all →
						</Link>
					</div>
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
										<TableCell class="text-muted">
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
				</Card>

				{/* Recent applications */}
				<Card>
					<div class="flex items-center justify-between border-b border-border px-5 py-4">
						<h3 class="text-base font-semibold leading-snug text-foreground">
							Recent applications
						</h3>
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
											{app.JobCompanySlug}
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

interface StatCardProps {
	label: string;
	value: string;
	delta: string;
	deltaUp?: boolean;
}

function StatCard(props: StatCardProps) {
	return (
		<Card>
			<div class="p-5">
				<p class="mb-2 text-xs font-semibold uppercase tracking-wider text-faint">
					{props.label}
				</p>
				<p class="font-mono text-3xl font-medium tabular-nums text-foreground">
					{props.value}
				</p>
				<p
					class={cn(
						"mt-1.5 text-xs",
						props.deltaUp ? "text-status-offer" : "text-faint",
					)}
				>
					{props.delta}
				</p>
			</div>
		</Card>
	);
}
