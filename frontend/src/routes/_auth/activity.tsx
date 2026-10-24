import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { format } from "date-fns";
import { createEffect, createSignal, For, on, Show } from "solid-js";
import { RangeMenu } from "@/components/activity/RangeMenu";
import { FilterPills, FilterSidebar } from "@/components/activity/RunFilters";
import { RunHistogram } from "@/components/activity/RunHistogram";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SearchField } from "@/components/SearchField";
import { SourceBadge } from "@/components/SourceBadge";
import { SourcesHealth } from "@/components/SourcesHealth";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useDebouncedTerm } from "@/hooks/useDebouncedTerm";
import {
	type ActivitySearch,
	activeFilterCount,
	formatDuration,
	newerPage,
	olderPage,
	pageSummary,
	parseActivitySearch,
	runFilters,
	setQuery,
} from "@/lib/scrapeRuns";
import { searchTitle } from "@/lib/searchTargets";
import { cn } from "@/lib/utils";
import type { ScrapeRun, ScrapeRunOutcome } from "@/types/scrapeRun";
import {
	scrapeRunSummaryQuery,
	scrapeRunsQuery,
	useScrapeRunSummary,
	useScrapeRuns,
	useSourceHealth,
} from "../../hooks/useScrapeRuns";
import { useSources } from "../../hooks/useSources";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/activity")({
	validateSearch: parseActivitySearch,
	loader: ({ location }) => {
		const { trail: _trail, ...params } = parseActivitySearch(location.search);
		return Promise.all([
			queryClient.prefetchQuery(scrapeRunsQuery(params)),
			queryClient.prefetchQuery(scrapeRunSummaryQuery(runFilters(params))),
		]);
	},
	component: ActivityPage,
});

const OUTCOMES: Record<
	ScrapeRunOutcome,
	{ label: string; dot: string; text: string }
> = {
	running: { label: "Running", dot: "bg-primary", text: "text-muted" },
	succeeded: { label: "Succeeded", dot: "bg-status-offer", text: "text-muted" },
	retried: { label: "Retried", dot: "bg-status-interview", text: "text-muted" },
	failed: {
		label: "Failed",
		dot: "bg-destructive",
		text: "text-destructive-strong",
	},
};

const STOP_REASONS = {
	"": "Still running",
	last_page: "Reached the last page",
	page_cap: "Stopped at the page cap",
	failed: "Gave up after repeated failures",
} as const;

const ROW_GRID =
	"grid grid-cols-[1.5rem_9.5rem_9rem_minmax(0,1fr)_3.5rem_5.5rem_4rem] items-center gap-x-3 2xl:grid-cols-[1.5rem_9.5rem_9rem_minmax(0,1fr)_3.5rem_5.5rem_4rem_4rem_4rem]";

function Count(props: { value: number; class?: string }) {
	return (
		<span
			class={cn(
				"text-right font-mono tabular-nums",
				props.value === 0 ? "text-faint" : "text-foreground",
				props.class,
			)}
		>
			{props.value === 0 ? "—" : props.value.toLocaleString()}
		</span>
	);
}

function RunRow(props: {
	run: ScrapeRun;
	sourceLabel: string;
	onlyThisSearch: () => void;
}) {
	const [open, setOpen] = createSignal(false);
	const outcome = () => OUTCOMES[props.run.Outcome];
	const title = () =>
		searchTitle({ Source: props.run.Source, Value: props.run.TargetValue });
	return (
		<li class="border-b border-border last:border-b-0">
			<button
				type="button"
				aria-expanded={open()}
				aria-label={`${title()}, ${outcome().label}`}
				onClick={() => setOpen((o) => !o)}
				class={cn(
					ROW_GRID,
					"w-full px-4 py-2.5 text-left text-data transition-colors hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary",
				)}
			>
				<Icon
					name={open() ? "chevronDown" : "chevronRight"}
					class="size-4 text-faint"
				/>
				<span class="flex items-center gap-1.5 font-mono tabular-nums text-foreground">
					{format(new Date(props.run.StartedAt), "d MMM, HH:mm")}
					<Show when={props.run.Trigger === "manual"}>
						<Icon
							name="user"
							class="size-3.5 text-faint"
							aria-label="Manual run"
						/>
					</Show>
				</span>
				<span class={cn("flex items-center gap-2", outcome().text)}>
					<span class={cn("size-2 rounded-full", outcome().dot)} />
					{outcome().label}
					<Show when={props.run.StopReason === "page_cap"}>
						<span class="text-xs text-status-interview">capped</span>
					</Show>
				</span>
				<span class="flex min-w-0 items-center gap-2">
					<SourceBadge source={props.sourceLabel} />
					<span class="truncate text-foreground">{title()}</span>
				</span>
				<Count value={props.run.New} class="font-semibold" />
				<Count value={props.run.Elsewhere} />
				<Count
					value={props.run.Retries}
					class={props.run.Retries > 0 ? "text-destructive-strong" : ""}
				/>
				<Count value={props.run.Pages} class="hidden 2xl:block" />
				<Count value={props.run.Repeat} class="hidden 2xl:block" />
			</button>
			<Show when={open()}>
				<div class="flex flex-col gap-4 px-4 pb-4 pl-12 text-data md:flex-row">
					<dl class="grid shrink-0 grid-cols-[5rem_1fr] gap-x-3 gap-y-1">
						<dt class="text-faint">Stopped</dt>
						<dd>{STOP_REASONS[props.run.StopReason]}</dd>
						<dt class="text-faint">Listings</dt>
						<dd>
							{props.run.Listings} across {props.run.Pages}{" "}
							{props.run.Pages === 1 ? "page" : "pages"}, {props.run.Repeat}{" "}
							{props.run.Repeat === 1 ? "repeat" : "repeats"}
						</dd>
						<dt class="text-faint">Trigger</dt>
						<dd class="capitalize">{props.run.Trigger}</dd>
						<dt class="text-faint">Duration</dt>
						<dd>
							{props.run.FinishedAt
								? formatDuration(props.run.StartedAt, props.run.FinishedAt)
								: "—"}
						</dd>
						<dt class="text-faint">Deferred</dt>
						<dd>{props.run.Deferred} rate-limit waits</dd>
					</dl>
					<div class="flex min-w-0 flex-1 flex-col gap-2">
						<Show when={props.run.Errors.length > 0}>
							<ul class="flex flex-col gap-1 rounded-md border border-border bg-surface p-3 font-mono text-xs text-destructive-strong">
								<For each={props.run.Errors}>
									{(err) => (
										<li class="break-words">
											<span class="text-faint">
												{format(new Date(err.At), "HH:mm:ss")} · p{err.Page} ·
												try {err.Attempt}
											</span>{" "}
											{err.Message}
										</li>
									)}
								</For>
							</ul>
						</Show>
						<div>
							<Button
								variant="outline"
								size="sm"
								onClick={props.onlyThisSearch}
							>
								Only this search
							</Button>
						</div>
					</div>
				</div>
			</Show>
		</li>
	);
}

function ActivityPage() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/activity" });
	const sources = useSources();
	const runs = useScrapeRuns(() => {
		const { trail: _trail, ...params } = search();
		return params;
	});
	const summary = useScrapeRunSummary(
		() => runFilters(search()),
		() => !search().before,
	);
	const go = (next: ActivitySearch) => navigate({ search: next });
	const [filtersOpen, setFiltersOpen] = createSignal(false);
	const [term, typeTerm] = useDebouncedTerm();
	const [text, setText] = createSignal(search().q ?? "");
	createEffect(
		on(
			term,
			(t) => {
				if (t !== (search().q ?? "")) go(setQuery(search(), t));
			},
			{ defer: true },
		),
	);
	const firstPage = (next: Partial<ActivitySearch>): ActivitySearch => ({
		...search(),
		...next,
		before: undefined,
		trail: undefined,
	});
	const sourceLabel = (name: string) =>
		sources.data?.find((s) => s.name === name)?.label ?? name;
	const health = useSourceHealth();
	let logTop: HTMLDivElement | undefined;
	const selectSource = (target: string) => {
		go({ target });
		logTop?.scrollIntoView({ behavior: "smooth" });
	};

	return (
		<div class="px-7 py-6">
			<QueryBoundary query={health} fallbackRows={4}>
				{(rows) => (
					<SourcesHealth
						rows={rows()}
						selected={search().target}
						sourceLabel={sourceLabel}
						onSelect={selectSource}
					/>
				)}
			</QueryBoundary>
			<div ref={logTop} class="scroll-mt-6" />
			<PageHeading
				title="Run log"
				subtitle="Every Search run with its counters and errors"
			>
				<span class="flex items-center gap-1.5 text-xs text-muted">
					<span
						class={cn(
							"size-2 rounded-full",
							search().before ? "bg-faint" : "bg-primary",
						)}
					/>
					{search().before ? "Paused" : "Live"}
				</span>
			</PageHeading>

			<div class="mb-3 flex flex-wrap items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					aria-expanded={filtersOpen()}
					onClick={() => setFiltersOpen((o) => !o)}
				>
					<Icon name="filter" class="size-3.5" />
					Filters
					<Show when={activeFilterCount(search()) > 0}>
						<span class="rounded-full bg-primary px-1.5 text-xs text-primary-foreground">
							{activeFilterCount(search())}
						</span>
					</Show>
				</Button>
				<SearchField
					id="run-search"
					label="Search runs"
					placeholder="Search names and error messages"
					value={text()}
					onInput={(value) => {
						setText(value);
						typeTerm(value);
					}}
					class="sm:max-w-80"
				/>
				<Show when={search().target}>
					<Button
						variant="outline"
						size="sm"
						onClick={() => go(firstPage({ target: undefined }))}
					>
						One search only
						<Icon name="x" class="size-3.5" />
					</Button>
				</Show>
				<div class="ml-auto flex items-center gap-3">
					<Show when={summary.data}>
						{(data) => (
							<span class="text-xs text-muted">
								<span class="font-mono tabular-nums text-foreground">
									{data().Total.toLocaleString()}
								</span>{" "}
								{data().Total === 1 ? "run" : "runs"}
							</span>
						)}
					</Show>
					<RangeMenu search={search()} onChange={go} />
				</div>
			</div>

			<Show when={activeFilterCount(search()) > 0}>
				<FilterPills
					search={search()}
					sourceLabel={sourceLabel}
					onChange={go}
				/>
			</Show>
			<Show when={summary.data}>
				{(data) => <RunHistogram summary={data()} />}
			</Show>

			<div class="flex items-start gap-6">
				<Show when={filtersOpen()}>
					<FilterSidebar
						search={search()}
						summary={summary.data}
						sourceLabel={sourceLabel}
						onChange={go}
					/>
				</Show>
				<div class="min-w-0 flex-1">
					<QueryBoundary query={runs} fallbackRows={8}>
						{(page) => (
							<Show
								when={page().Items.length > 0}
								fallback={
									<Card class="p-10 text-center">
										<p class="text-sm text-muted">
											{activeFilterCount(search()) > 0 || search().q
												? "No runs match these filters."
												: "No runs yet."}
										</p>
									</Card>
								}
							>
								<div class="overflow-hidden rounded-xl border border-border bg-surface">
									<div
										class={cn(
											ROW_GRID,
											"border-b border-border bg-surface-muted px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-faint",
										)}
									>
										<span />
										<span>Started</span>
										<span>Outcome</span>
										<span>Search or board</span>
										<span class="text-right">New</span>
										<span class="text-right">Elsewhere</span>
										<span class="text-right">Retries</span>
										<span class="hidden text-right 2xl:block">Pages</span>
										<span class="hidden text-right 2xl:block">Repeat</span>
									</div>
									<ul>
										<For each={page().Items}>
											{(run) => (
												<RunRow
													run={run}
													sourceLabel={sourceLabel(run.Source)}
													onlyThisSearch={() =>
														go(firstPage({ target: run.TargetID }))
													}
												/>
											)}
										</For>
									</ul>
									<div class="flex items-center justify-end gap-2 border-t border-border px-4 py-2.5">
										<Show when={summary.data}>
											{(data) => (
												<span class="mr-auto font-mono text-xs tabular-nums text-muted">
													{pageSummary(
														search(),
														page().Items.length,
														data().Total,
													)}
												</span>
											)}
										</Show>
										<Button
											variant="outline"
											size="sm"
											disabled={!search().before}
											onClick={() => go(firstPage({}))}
										>
											Newest
										</Button>
										<Button
											variant="outline"
											size="sm"
											disabled={!search().before}
											onClick={() => go(newerPage(search()))}
										>
											Newer
										</Button>
										<Button
											variant="outline"
											size="sm"
											disabled={!page().NextCursor}
											onClick={() => {
												const next = page().NextCursor;
												if (next) go(olderPage(search(), next));
											}}
										>
											Older
										</Button>
									</div>
								</div>
							</Show>
						)}
					</QueryBoundary>
				</div>
			</div>
		</div>
	);
}
