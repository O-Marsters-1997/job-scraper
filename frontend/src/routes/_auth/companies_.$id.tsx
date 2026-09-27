import { createFileRoute, Link } from "@tanstack/solid-router";
import {
	createEffect,
	createMemo,
	createSignal,
	For,
	onCleanup,
	Show,
} from "solid-js";
import { createJobColumns } from "@/components/jobs/columns";
import { JobsDataTable } from "@/components/jobs/JobsDataTable";
import { SourceBadge } from "@/components/SourceBadge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import { formatDate } from "@/lib/datetime";
import {
	applyJobFilters,
	DEFAULT_FILTERS,
	filterCompanyJobs,
	sourceOptions,
} from "@/lib/jobFilters";
import {
	companiesQueryOptions,
	useAddCompanyBoard,
	useCompanies,
	useCompanyBoards,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { useAllJobs } from "../../hooks/useJobs";
import { queryClient } from "../../lib/queryClient";
import type { CompanyBoard } from "../../types/company";

function boardURLFor(board: CompanyBoard): string {
	switch (board.Source) {
		case "greenhouse":
			return `https://boards.greenhouse.io/${board.BoardToken}`;
		case "lever":
			return `https://jobs.lever.co/${board.BoardToken}`;
		case "ashby":
			return `https://jobs.ashbyhq.com/${board.BoardToken}`;
		case "workable":
			return `https://apply.workable.com/${board.BoardToken}`;
		case "recruitee":
			return `https://${board.BoardToken}.recruitee.com`;
		case "personio":
			return `https://${board.BoardToken}.jobs.personio.de`;
		default:
			return "";
	}
}

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

const BOARD_CHECK_POLL_MS = 2000;
const BOARD_CHECK_TIMEOUT_MS = 30_000;

export const Route = createFileRoute("/_auth/companies_/$id")({
	loader: () => queryClient.ensureQueryData(companiesQueryOptions),
	component: CompanyDetailPage,
});

function CompanyDetailPage() {
	const params = Route.useParams();
	const companiesQuery = useCompanies();
	const jobsQuery = useAllJobs();
	const trackMutation = useSetCompanyTracking();
	const [checkingBoardID, setCheckingBoardID] = createSignal<string>();
	const boardsQuery = useCompanyBoards(
		() => params().id,
		() => (checkingBoardID() ? BOARD_CHECK_POLL_MS : false),
	);
	const addBoardMutation = useAddCompanyBoard();
	const [boardURL, setBoardURL] = createSignal("");
	const [boardMessage, setBoardMessage] = createSignal("");
	let checkTimeout: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(checkTimeout));
	createEffect(() => {
		const id = checkingBoardID();
		const board = boardsQuery.data?.find((b) => b.ID === id);
		if (board?.Status !== "verified") return;
		clearTimeout(checkTimeout);
		setCheckingBoardID(undefined);
		setBoardMessage("Board verified.");
	});
	const saveBoard = async (url: string) => {
		setBoardMessage("");
		clearTimeout(checkTimeout);
		try {
			const board = await addBoardMutation.mutateAsync({
				id: params().id,
				url,
				confirm: true,
			});
			setBoardURL("");
			if (board.Status === "verified") {
				setBoardMessage("Board verified.");
				return;
			}
			setBoardMessage("Checking the board…");
			setCheckingBoardID(board.ID);
			checkTimeout = setTimeout(() => {
				setCheckingBoardID(undefined);
				setBoardMessage(
					"Verification failed; retry when the board is available.",
				);
			}, BOARD_CHECK_TIMEOUT_MS);
		} catch (error) {
			setBoardMessage(
				error instanceof Error ? error.message : "Could not add board.",
			);
		}
	};

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

						<div class="grid grid-cols-[1fr_284px] items-start gap-4">
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
											columns={columns}
											data={companyJobs()}
											filters={filters()}
											onChange={setFilterPatch}
											sourceOptions={sourceOptions(jobsForCompany())}
										/>
									</Show>
								</CardContent>
							</Card>

							<div class="sticky top-0 flex flex-col gap-3">
								<Card>
									<CardHeader class="pb-2">
										<CardTitle>Boards</CardTitle>
									</CardHeader>
									<CardContent class="gap-3">
										<Show when={boardsQuery.isError}>
											<p class="text-sm text-destructive-strong">
												Could not load boards.
											</p>
										</Show>
										<Show when={boardsQuery.isPending}>
											<p class="text-sm text-muted">Loading boards…</p>
										</Show>
										<Show when={!boardsQuery.isPending && !boardsQuery.isError}>
											<Show
												when={(boardsQuery.data ?? []).length > 0}
												fallback={
													<p class="text-sm text-faint">
														No boards linked yet.
													</p>
												}
											>
												<For each={boardsQuery.data ?? []}>
													{(board) => (
														<div class="flex items-start justify-between gap-2 border-b border-border pb-2 text-xs">
															<div class="min-w-0">
																<p class="font-medium text-foreground">
																	{board.Source}
																</p>
																<p class="break-all font-mono text-faint">
																	{board.BoardToken}
																</p>
																<Show when={board.LastCompletedAt}>
																	{(checked) => (
																		<p class="text-faint">
																			Last checked{" "}
																			{new Date(checked()).toLocaleString()}
																		</p>
																	)}
																</Show>
															</div>
															<span class="capitalize text-muted">
																{board.Status}
															</span>
														</div>
													)}
												</For>
											</Show>
										</Show>
										<form
											onSubmit={(event) => {
												event.preventDefault();
												void saveBoard(boardURL().trim());
											}}
											class="flex flex-col gap-2"
										>
											<label
												for="company-board-url"
												class="text-xs font-medium text-foreground"
											>
												ATS board URL
											</label>
											<Input
												id="company-board-url"
												type="url"
												required
												value={boardURL()}
												onInput={(event) =>
													setBoardURL(event.currentTarget.value)
												}
												placeholder="https://boards.greenhouse.io/acme"
											/>
											<Button
												type="submit"
												size="sm"
												disabled={addBoardMutation.isPending}
											>
												Add and verify board
											</Button>
											<Show when={boardMessage()}>
												<output class="text-xs text-muted">
													{boardMessage()}
												</output>
											</Show>
										</form>
										<For
											each={(boardsQuery.data ?? []).filter(
												(board) => board.Status === "candidate",
											)}
										>
											{(board) => (
												<Button
													type="button"
													size="sm"
													variant="outline"
													disabled={addBoardMutation.isPending}
													onClick={() => saveBoard(boardURLFor(board))}
												>
													Retry {board.BoardToken}
												</Button>
											)}
										</For>
									</CardContent>
								</Card>
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
													if (!opt) return;
													trackMutation.mutate({
														id: c().ID,
														enabled: c().Tracked,
														checkIntervalMinutes: opt.minutes,
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
