import { createFileRoute, Link } from "@tanstack/solid-router";
import type { JSX } from "solid-js";
import { createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { JobActionsMenu } from "@/components/jobs/JobActionsMenu";
import JobDescription from "@/components/jobs/JobDescription";
import { SuitabilityPanel } from "@/components/jobs/SuitabilityPanel";
import { TrackApplicationDialog } from "@/components/jobs/TrackApplicationDialog";
import { SourceBadge } from "@/components/SourceBadge";
import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/datetime";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import { titleCase } from "@/lib/utils";
import type {
	ApplicationWithDetails,
	JobApplicationSummary,
} from "@/types/application";
import {
	applicationsQueryOptions,
	useApplications,
} from "../../hooks/useApplications";
import { jobQueryOptions, useJob } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/jobs_/$id")({
	loader: ({ params }) =>
		Promise.all([
			queryClient.ensureQueryData(jobQueryOptions(params.id)),
			queryClient.ensureQueryData(applicationsQueryOptions()),
		]),
	component: JobDetailPage,
});

function workStyle(daysInOffice: number | null | undefined): string | null {
	if (daysInOffice === null || daysInOffice === undefined) return null;
	if (daysInOffice === 0) return "Remote";
	if (daysInOffice >= 5) return "On-site";
	return `Hybrid · ${daysInOffice}d/wk`;
}

function MetaItem(props: { icon: JSX.Element; children: JSX.Element }) {
	return (
		<span class="flex items-center gap-1 text-xs text-muted">
			<span class="text-faint">{props.icon}</span>
			{props.children}
		</span>
	);
}

function FactRow(props: {
	label: string;
	value?: string;
	children?: JSX.Element;
}) {
	return (
		<div class="flex items-start justify-between gap-4">
			<span class="shrink-0 text-xs text-faint">{props.label}</span>
			<span class="text-right text-xs text-foreground">
				{props.children ?? props.value}
			</span>
		</div>
	);
}

function JobDetailPage() {
	const params = Route.useParams();
	const jobsQuery = useJob(() => params().id);
	const appsQuery = useApplications();

	const job = () => jobsQuery.data;
	const app = (): ApplicationWithDetails | undefined =>
		appsQuery.data?.find((a) => a.JobID === params().id);
	const appSummary = (): JobApplicationSummary | undefined => {
		const a = app();
		if (!a) return undefined;
		return {
			ApplicationID: a.ID,
			StatusID: a.StatusID,
			StatusName: a.StatusName,
			StatusColour: a.StatusColour,
		};
	};

	const [modalOpen, setModalOpen] = createSignal(false);

	const openTrack = () => setModalOpen(true);
	const openEdit = () => {
		if (!app()) return;
		setModalOpen(true);
	};

	const existingApp = () => {
		const a = app();
		if (!a) return undefined;
		return {
			id: a.ID,
			statusId: a.StatusID,
			notes: a.Notes,
			appliedAt: a.AppliedAt,
			salaryInfo: a.SalaryInfo,
		};
	};

	return (
		<Show
			when={!jobsQuery.isPending}
			fallback={<div class="px-7 py-6 text-sm text-muted">Loading…</div>}
		>
			<Show
				when={job()}
				fallback={
					<div class="flex h-[calc(100vh-14rem)] flex-col items-center justify-center gap-4 text-center">
						<Icon
							name="zoomIn"
							size={40}
							strokeWidth={1.5}
							class="text-faint"
						/>
						<div>
							<p class="text-base font-semibold text-foreground">
								Job not found
							</p>
							<p class="mt-1 text-sm text-muted">
								No job with ID{" "}
								<code class="rounded bg-surface-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
									{params().id}
								</code>
							</p>
						</div>
						<Button as={Link} to="/jobs" variant="outline" size="sm">
							← Back to Jobs
						</Button>
					</div>
				}
			>
				{(j) => (
					<div class="px-7 py-6 pb-16">
						<Card class="mb-4">
							<CardContent class="pt-5">
								<div class="flex gap-4">
									<div class="flex h-[52px] w-[52px] shrink-0 items-center justify-center rounded-xl bg-accent-subtle text-base font-semibold text-accent-text">
										{titleCase(j().CompanySlug).slice(0, 2)}
									</div>

									<div class="min-w-0 flex-1">
										<h1 class="text-xl font-bold tracking-tight text-foreground">
											{j().Title}
										</h1>
										<p class="mt-0.5 text-sm text-muted">
											{titleCase(j().CompanySlug)} · {j().Location}
										</p>

										<Show
											when={
												workStyle(j().DaysInOffice) ||
												j().EmploymentType ||
												j().SalaryRange ||
												j().ExperienceLevel
											}
										>
											<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1">
												<Show when={workStyle(j().DaysInOffice)}>
													{(ws) => (
														<MetaItem icon={<Icon name="home" size={12} />}>
															{ws()}
														</MetaItem>
													)}
												</Show>
												<Show when={j().EmploymentType}>
													{(et) => (
														<MetaItem
															icon={<Icon name="briefcase" size={12} />}
														>
															{et()}
														</MetaItem>
													)}
												</Show>
												<Show when={j().SalaryRange}>
													{(sr) => (
														<MetaItem icon={<Icon name="dollar" size={12} />}>
															<span class="font-mono tabular-nums">{sr()}</span>
														</MetaItem>
													)}
												</Show>
												<Show when={j().ExperienceLevel}>
													{(el) => (
														<MetaItem icon={<Icon name="users" size={12} />}>
															{el()}
														</MetaItem>
													)}
												</Show>
											</div>
										</Show>

										<div class="mt-3 flex flex-wrap items-center gap-2">
											<Show when={app()?.StatusName}>
												{(name) => (
													<StatusBadge
														name={name()}
														colour={
															app()?.StatusColour || STATUS_FALLBACK_COLOUR
														}
													/>
												)}
											</Show>
											<SourceBadge source={j().Source} />
											<span class="font-mono text-xs tabular-nums text-faint">
												{formatDate(j().ScrapedAt)}
											</span>
										</div>
									</div>

									<div class="shrink-0">
										<JobActionsMenu
											job={j()}
											appSummary={appSummary()}
											onTrack={openTrack}
											onEdit={openEdit}
										/>
									</div>
								</div>
							</CardContent>
						</Card>

						<div class="grid grid-cols-[1fr_284px] items-start gap-4">
							<Card>
								<CardHeader>
									<CardTitle>Description</CardTitle>
								</CardHeader>
								<CardContent class="gap-0">
									<Show
										when={j().Description}
										fallback={
											<p class="text-sm text-faint">
												No description captured.{" "}
												<a
													href={j().URL}
													target="_blank"
													rel="noopener noreferrer"
													class="text-primary transition-colors hover:underline"
												>
													View the original listing ↗
												</a>
											</p>
										}
									>
										{(desc) => <JobDescription html={desc()} />}
									</Show>
								</CardContent>
							</Card>

							<div class="sticky top-0 flex flex-col gap-3">
								<Card>
									<CardHeader class="pb-2">
										<CardTitle>Details</CardTitle>
									</CardHeader>
									<CardContent class="gap-2.5">
										<FactRow
											label="Company"
											value={titleCase(j().CompanySlug)}
										/>
										<Show when={j().Location}>
											{(loc) => <FactRow label="Location" value={loc()} />}
										</Show>
										<Show when={workStyle(j().DaysInOffice)}>
											{(ws) => <FactRow label="Work style" value={ws()} />}
										</Show>
										<FactRow label="Source">
											<SourceBadge source={j().Source} />
										</FactRow>
										<FactRow label="Scraped">
											<span class="font-mono text-xs tabular-nums text-foreground">
												{formatDate(j().ScrapedAt)}
											</span>
										</FactRow>
										<Show when={j().EmploymentType}>
											{(et) => <FactRow label="Employment" value={et()} />}
										</Show>
										<Show when={j().SalaryRange}>
											{(sr) => (
												<FactRow label="Salary">
													<span class="font-mono text-xs tabular-nums text-foreground">
														{sr()}
													</span>
												</FactRow>
											)}
										</Show>
										<Show when={j().ExperienceLevel}>
											{(el) => <FactRow label="Experience" value={el()} />}
										</Show>
										<Show when={j().TeamName}>
											{(tn) => <FactRow label="Team" value={tn()} />}
										</Show>
										<Show when={j().CompanySize}>
											{(cs) => <FactRow label="Company size" value={cs()} />}
										</Show>
										<div class="pt-1">
											<Button
												as="a"
												href={j().URL}
												target="_blank"
												rel="noopener noreferrer"
												variant="secondary"
												size="sm"
												class="w-full"
											>
												View listing ↗
											</Button>
										</div>
									</CardContent>
								</Card>

								<Show when={(j().Skills?.length ?? 0) > 0}>
									<Card>
										<CardHeader class="pb-2">
											<CardTitle>Skills & technologies</CardTitle>
										</CardHeader>
										<CardContent>
											<div class="flex flex-wrap gap-1.5">
												<For each={j().Skills}>
													{(skill) => (
														<span class="rounded-full border border-border bg-surface px-2.5 py-0.5 text-xs text-muted">
															{skill}
														</span>
													)}
												</For>
											</div>
										</CardContent>
									</Card>
								</Show>

								<SuitabilityPanel job={j()} />

								<Card>
									<CardHeader class="pb-2">
										<CardTitle>Application</CardTitle>
									</CardHeader>
									<Show
										when={app()}
										fallback={
											<CardContent class="gap-3">
												<p class="text-xs text-faint">Not yet tracked</p>
												<Button class="w-full" onClick={openTrack}>
													Track application
												</Button>
											</CardContent>
										}
									>
										{(a) => (
											<CardContent class="gap-2.5">
												<div>
													<StatusBadge
														name={a().StatusName}
														colour={a().StatusColour || STATUS_FALLBACK_COLOUR}
													/>
												</div>
												<Show when={a().AppliedAt}>
													{(date) => (
														<FactRow label="Applied">
															<span class="font-mono text-xs tabular-nums text-foreground">
																{formatDate(date())}
															</span>
														</FactRow>
													)}
												</Show>
												<Show when={a().SalaryInfo}>
													{(salary) => (
														<FactRow label="Salary">
															<span class="font-mono text-xs tabular-nums text-foreground">
																{salary()}
															</span>
														</FactRow>
													)}
												</Show>
												<Show when={a().Notes}>
													{(notes) => (
														<p class="rounded-md bg-surface-muted px-3 py-2 text-xs text-muted">
															{notes()}
														</p>
													)}
												</Show>
												<div class="pt-0.5">
													<Button
														variant="outline"
														size="sm"
														class="w-full"
														onClick={openEdit}
													>
														Edit
													</Button>
												</div>
											</CardContent>
										)}
									</Show>
								</Card>
							</div>
						</div>

						<TrackApplicationDialog
							open={modalOpen()}
							onOpenChange={setModalOpen}
							job={j()}
							existingApp={existingApp()}
						/>
					</div>
				)}
			</Show>
		</Show>
	);
}
