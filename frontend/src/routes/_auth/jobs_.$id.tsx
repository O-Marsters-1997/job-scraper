import { createFileRoute, Link } from "@tanstack/solid-router";
import type { JSX } from "solid-js";
import { createSignal, For, Show } from "solid-js";
import { titleCase } from "@/components/jobs/columns";
import { JobActionsMenu } from "@/components/jobs/JobActionsMenu";
import { TrackApplicationDialog } from "@/components/jobs/TrackApplicationDialog";
import { SourceBadge } from "@/components/SourceBadge";
import { StatusBadge } from "@/components/StatusBadge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/datetime";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import type {
	ApplicationWithDetails,
	JobApplicationSummary,
} from "@/types/application";
import {
	applicationsQueryOptions,
	useApplications,
} from "../../hooks/useApplications";
import { jobsQueryOptions, useJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/jobs_/$id")({
	loader: () =>
		Promise.all([
			queryClient.ensureQueryData(jobsQueryOptions),
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

function SuitabilityPanel(props: { job: import("@/types/job").Job }) {
	const score = () => props.job.SuitabilityScore;
	const reasoning = () => props.job.Reasoning ?? null;
	const matched = () => props.job.Matched ?? [];
	const missing = () => props.job.Missing ?? [];

	// scored — has score + reasoning
	const isScored = () => score() != null && reasoning() != null;
	// legacy scored — has score but no reasoning
	const isLegacy = () => score() != null && reasoning() == null;
	// pending — no score (skipped state handled in ticket #104)
	const isPending = () => score() == null;

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Suitability</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<Show when={isScored()}>
					{/* Score display */}
					<div class="flex items-baseline gap-1">
						<span class="font-mono text-lg font-semibold tabular-nums text-foreground">
							{score()}
						</span>
						<span class="text-xs text-faint">/ 100</span>
					</div>

					{/* Rationale */}
					<p class="rounded-md bg-surface-muted px-3 py-2 text-xs leading-relaxed text-muted">
						{reasoning()}
					</p>

					{/* Matched chips */}
					<Show when={matched().length > 0}>
						<div class="flex flex-col gap-1.5">
							<span class="text-xs font-medium text-faint">Matched</span>
							<div class="flex flex-wrap gap-1">
								<For each={matched()}>
									{(item) => (
										<span
											class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
											style={{
												background: "color-mix(in srgb, #059669 12%, white)",
												color: "color-mix(in srgb, #059669 80%, black)",
												border:
													"1px solid color-mix(in srgb, #059669 28%, white)",
											}}
										>
											{item}
										</span>
									)}
								</For>
							</div>
						</div>
					</Show>

					{/* Missing chips */}
					<Show when={missing().length > 0}>
						<div class="flex flex-col gap-1.5">
							<span class="text-xs font-medium text-faint">Missing</span>
							<div class="flex flex-wrap gap-1">
								<For each={missing()}>
									{(item) => <Badge variant="secondary">{item}</Badge>}
								</For>
							</div>
						</div>
					</Show>
				</Show>

				<Show when={isLegacy()}>
					<div class="flex items-baseline gap-1">
						<span class="font-mono text-lg font-semibold tabular-nums text-foreground">
							{score()}
						</span>
						<span class="text-xs text-faint">/ 100</span>
					</div>
					<p class="text-xs text-faint">
						No reasoning captured for this score.
					</p>
				</Show>

				<Show when={isPending()}>
					<p class="text-xs text-faint">Not yet scored.</p>
				</Show>
			</CardContent>
		</Card>
	);
}

function JobDetailPage() {
	const params = Route.useParams();
	const jobsQuery = useJobs();
	const appsQuery = useApplications();

	const job = () => jobsQuery.data?.find((j) => j.ID === params().id);
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
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="40"
							height="40"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="1.5"
							stroke-linecap="round"
							stroke-linejoin="round"
							class="text-faint"
							aria-hidden="true"
						>
							<circle cx="11" cy="11" r="8" />
							<line x1="21" y1="21" x2="16.65" y2="16.65" />
							<line x1="11" y1="8" x2="11" y2="14" />
							<line x1="8" y1="11" x2="14" y2="11" />
						</svg>
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
						<Link to="/jobs">
							<Button variant="outline" size="sm">
								← Back to Jobs
							</Button>
						</Link>
					</div>
				}
			>
				{(j) => (
					<div class="px-7 py-6 pb-16">
						{/* ── Hero card ───────────────────────────────────── */}
						<Card class="mb-4">
							<CardContent class="pt-5">
								<div class="flex gap-4">
									{/* Company avatar */}
									<div class="flex h-[52px] w-[52px] shrink-0 items-center justify-center rounded-xl bg-accent-subtle text-base font-semibold text-accent-text">
										{titleCase(j().CompanySlug).slice(0, 2)}
									</div>

									{/* Title + meta */}
									<div class="min-w-0 flex-1">
										<h1 class="text-xl font-bold tracking-tight text-foreground">
											{j().Title}
										</h1>
										<p class="mt-0.5 text-sm text-muted">
											{titleCase(j().CompanySlug)} · {j().Location}
										</p>

										{/* Meta strip — only rendered items */}
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
														<MetaItem
															icon={
																<svg
																	xmlns="http://www.w3.org/2000/svg"
																	width="12"
																	height="12"
																	viewBox="0 0 24 24"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2"
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	aria-hidden="true"
																>
																	<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
																	<polyline points="9 22 9 12 15 12 15 22" />
																</svg>
															}
														>
															{ws()}
														</MetaItem>
													)}
												</Show>
												<Show when={j().EmploymentType}>
													{(et) => (
														<MetaItem
															icon={
																<svg
																	xmlns="http://www.w3.org/2000/svg"
																	width="12"
																	height="12"
																	viewBox="0 0 24 24"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2"
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	aria-hidden="true"
																>
																	<rect
																		x="2"
																		y="7"
																		width="20"
																		height="14"
																		rx="2"
																		ry="2"
																	/>
																	<path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
																</svg>
															}
														>
															{et()}
														</MetaItem>
													)}
												</Show>
												<Show when={j().SalaryRange}>
													{(sr) => (
														<MetaItem
															icon={
																<svg
																	xmlns="http://www.w3.org/2000/svg"
																	width="12"
																	height="12"
																	viewBox="0 0 24 24"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2"
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	aria-hidden="true"
																>
																	<line x1="12" y1="1" x2="12" y2="23" />
																	<path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
																</svg>
															}
														>
															<span class="font-mono tabular-nums">{sr()}</span>
														</MetaItem>
													)}
												</Show>
												<Show when={j().ExperienceLevel}>
													{(el) => (
														<MetaItem
															icon={
																<svg
																	xmlns="http://www.w3.org/2000/svg"
																	width="12"
																	height="12"
																	viewBox="0 0 24 24"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2"
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	aria-hidden="true"
																>
																	<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
																	<circle cx="9" cy="7" r="4" />
																	<path d="M23 21v-2a4 4 0 0 0-3-3.87" />
																	<path d="M16 3.13a4 4 0 0 1 0 7.75" />
																</svg>
															}
														>
															{el()}
														</MetaItem>
													)}
												</Show>
											</div>
										</Show>

										{/* Badges row */}
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

									{/* Actions menu */}
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

						{/* ── Two-column body ─────────────────────────────── */}
						<div class="grid grid-cols-[1fr_284px] items-start gap-4">
							{/* Left: Description */}
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
										{(desc) => (
											<div class="flex flex-col gap-3 text-sm leading-relaxed text-foreground">
												<For each={desc().split("\n\n")}>
													{(para) => <p>{para}</p>}
												</For>
											</div>
										)}
									</Show>
								</CardContent>
							</Card>

							{/* Right: sticky sidebar */}
							<div class="sticky top-0 flex flex-col gap-3">
								{/* Details card */}
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
											<a
												href={j().URL}
												target="_blank"
												rel="noopener noreferrer"
												class="block"
											>
												<Button variant="secondary" size="sm" class="w-full">
													View listing ↗
												</Button>
											</a>
										</div>
									</CardContent>
								</Card>

								{/* Skills card — only when present */}
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

								{/* Suitability card */}
								<SuitabilityPanel job={j()} />

								{/* Application card */}
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
