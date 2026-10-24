import { Link } from "@tanstack/solid-router";
import { format, formatDistanceToNow } from "date-fns";
import {
	createEffect,
	createMemo,
	createSignal,
	For,
	onCleanup,
	onMount,
	Show,
} from "solid-js";
import { RangeMenu } from "@/components/activity/RangeMenu";
import { Icon } from "@/components/Icon";
import { SourceBadge } from "@/components/SourceBadge";
import {
	DEFAULT_RANGE,
	formatDuration,
	groupPollsByDay,
	type HistorySearch,
	RANGE_PRESETS,
} from "@/lib/scrapeRuns";
import { cn } from "@/lib/utils";
import type { ScrapeRun } from "@/types/scrapeRun";
import type { ScrapeRunFilters } from "../../../api/scrapeRuns";
import { useCompanyBoards } from "../../../hooks/useCompanies";
import {
	useScrapeRunPages,
	useScrapeRunSummary,
} from "../../../hooks/useScrapeRuns";

const ROW_GRID =
	"grid grid-cols-[1rem_3.5rem_minmax(0,1fr)_3rem_4rem_4rem] items-center gap-x-3";

const CHANGE_STYLE = {
	new: { label: "New", class: "text-status-offer" },
	removed: { label: "Removed", class: "text-destructive-strong" },
	reopened: { label: "Reopened", class: "text-muted" },
} as const;

function Count(props: { value: number; sign?: string; class?: string }) {
	return (
		<span
			class={cn(
				"text-right font-mono tabular-nums",
				props.value === 0 ? "text-faint" : (props.class ?? "text-foreground"),
			)}
		>
			{props.value === 0 ? "—" : `${props.sign ?? ""}${props.value}`}
		</span>
	);
}

function rangeLabel(range: HistorySearch): string {
	if (range.from && range.to) return "in range";
	const preset = (range.range ?? DEFAULT_RANGE) as keyof typeof RANGE_PRESETS;
	return RANGE_PRESETS[preset].toLowerCase();
}

export function CompanyHistory(props: {
	companyId: string;
	intervalMinutes: number;
	range: HistorySearch;
	onRangeChange: (next: HistorySearch) => void;
}) {
	const filters = (): ScrapeRunFilters => ({
		kind: "board",
		company: props.companyId,
		range: props.range.range,
		from: props.range.from,
		to: props.range.to,
	});
	const query = useScrapeRunPages(filters);
	const summary = useScrapeRunSummary(filters, () => false);
	const boards = useCompanyBoards(() => props.companyId);
	const tokenFor = (boardId: string) =>
		boards.data?.find((b) => b.ID === boardId)?.BoardToken ?? "";
	const board = () => boards.data?.find((b) => b.Status === "verified");
	const days = createMemo(() =>
		groupPollsByDay(query.data?.pages.flatMap((p) => p.Items) ?? []),
	);

	let sentinel: HTMLDivElement | undefined;
	let observer: IntersectionObserver | undefined;
	onMount(() => {
		observer = new IntersectionObserver(
			(entries) => {
				if (
					entries[0]?.isIntersecting &&
					query.hasNextPage &&
					!query.isFetchingNextPage
				)
					query.fetchNextPage();
			},
			{ rootMargin: "400px" },
		);
		onCleanup(() => observer?.disconnect());
	});
	createEffect(() => {
		query.data?.pages.length;
		if (!sentinel || !observer) return;
		observer.unobserve(sentinel);
		observer.observe(sentinel);
	});

	return (
		<div>
			<div class="mb-3 flex items-center gap-2">
				<span class="text-xs text-muted">
					Polled every{" "}
					{props.intervalMinutes % 60 === 0
						? `${props.intervalMinutes / 60}h`
						: `${props.intervalMinutes}m`}
					<Show when={board()}>
						{(b) => (
							<>
								{" "}
								· {b().Source}/{b().BoardToken}
							</>
						)}
					</Show>
				</span>
				<div class="ml-auto">
					<RangeMenu search={props.range} onChange={props.onRangeChange} />
				</div>
			</div>
			<Show when={summary.data}>
				{(s) => (
					<dl class="mb-6 grid grid-cols-2 gap-4 rounded-xl border border-border bg-surface p-4 text-sm sm:grid-cols-5">
						<div>
							<dt class="text-xs text-faint">
								Polls {rangeLabel(props.range)}
							</dt>
							<dd class="font-semibold text-foreground">
								{s().Total.toLocaleString()}
							</dd>
						</div>
						<div>
							<dt class="text-xs text-faint">New jobs</dt>
							<dd class="font-semibold text-status-offer">+{s().New}</dd>
						</div>
						<div>
							<dt class="text-xs text-faint">Removed</dt>
							<dd class="font-semibold text-destructive-strong">
								−{s().Removed}
							</dd>
						</div>
						<div>
							<dt class="text-xs text-faint">Failed</dt>
							<dd
								class={cn(
									"font-semibold",
									s().Facets.outcome?.failed
										? "text-destructive-strong"
										: "text-foreground",
								)}
							>
								{s().Facets.outcome?.failed ?? 0}
							</dd>
						</div>
						<div>
							<dt class="text-xs text-faint">Last polled</dt>
							<dd class="font-semibold text-foreground">
								<Show when={s().LastStartedAt} fallback="—">
									{(at) =>
										formatDistanceToNow(new Date(at()), { addSuffix: true })
									}
								</Show>
							</dd>
						</div>
					</dl>
				)}
			</Show>

			<Show when={query.isError}>
				<p class="text-sm text-destructive-strong">Couldn't load polls.</p>
			</Show>
			<Show
				when={!query.isPending}
				fallback={<p class="text-sm text-faint">Loading polls…</p>}
			>
				<Show
					when={days().length > 0}
					fallback={
						<p class="rounded-xl border border-border bg-surface p-10 text-center text-sm text-muted">
							No polls in this range.
						</p>
					}
				>
					<For each={days()}>
						{(day) => (
							<section class="mb-6">
								<h3 class="mb-2 text-xs font-semibold uppercase tracking-wide text-faint">
									{format(new Date(day.day), "EEEE d MMMM")}
								</h3>
								<ol class="relative ml-2 border-l border-border">
									<For each={day.entries}>
										{(entry) =>
											entry.kind === "quiet" ? (
												<QuietEntry runs={entry.runs} />
											) : (
												<PollEntry run={entry.run} token={tokenFor} />
											)
										}
									</For>
								</ol>
							</section>
						)}
					</For>
				</Show>
			</Show>
			<div ref={sentinel} class="py-4 text-center text-xs text-faint">
				<Show when={query.isFetchingNextPage}>Loading older polls…</Show>
				<Show when={!query.isPending && !query.hasNextPage && days().length}>
					No older polls in this range.
				</Show>
			</div>
		</div>
	);
}

function QuietEntry(props: { runs: ScrapeRun[] }) {
	const first = () => props.runs.at(-1)?.StartedAt ?? "";
	const last = () => props.runs[0]?.StartedAt ?? "";
	return (
		<li class="relative py-1.5 pl-5 text-xs text-faint">
			<span class="absolute top-3 -left-[3px] size-1.5 rounded-full bg-border-strong" />
			<span class="font-mono tabular-nums">
				{format(new Date(first()), "HH:mm")}
				<Show when={props.runs.length > 1}>
					–{format(new Date(last()), "HH:mm")}
				</Show>
			</span>{" "}
			· {props.runs.length} {props.runs.length === 1 ? "poll" : "polls"}, no
			changes · {props.runs[0]?.Listings} listings
		</li>
	);
}

function PollEntry(props: {
	run: ScrapeRun;
	token: (boardId: string) => string;
}) {
	const [open, setOpen] = createSignal(false);
	const failed = () => props.run.Outcome === "failed";
	const headline = () => {
		switch (props.run.Outcome) {
			case "failed":
				return "Poll failed";
			case "running":
				return "Polling…";
			case "retried":
				return "Poll retried";
			case "succeeded":
				return "Poll";
		}
	};
	return (
		<li class="relative py-1.5 pl-5">
			<span
				class={cn(
					"absolute top-[1.15rem] -left-[5px] size-2.5 rounded-full ring-4 ring-background",
					failed() ? "bg-destructive" : "bg-status-offer",
				)}
			/>
			<div class="overflow-hidden rounded-lg border border-border bg-surface">
				<button
					type="button"
					aria-expanded={open()}
					onClick={() => setOpen((o) => !o)}
					class={cn(
						ROW_GRID,
						"w-full px-3 py-2.5 text-left text-data transition-colors hover:bg-surface-muted",
					)}
				>
					<Icon
						name={open() ? "chevronDown" : "chevronRight"}
						class="size-4 text-faint"
					/>
					<span class="flex items-center gap-1 font-mono tabular-nums text-foreground">
						{format(new Date(props.run.StartedAt), "HH:mm")}
						<Show when={props.run.Trigger === "manual"}>
							<Icon
								name="user"
								class="size-3.5 text-faint"
								aria-label="Manual poll"
							/>
						</Show>
					</span>
					<span
						class={cn(
							"truncate",
							failed() ? "text-destructive-strong" : "text-foreground",
						)}
					>
						{headline()}
						<Show when={props.run.Changes.length > 0}>
							<span class="text-muted">
								{" "}
								· {props.run.Changes.map((c) => c.Title).join(", ")}
							</span>
						</Show>
					</span>
					<Count value={props.run.New} sign="+" class="text-status-offer" />
					<Count
						value={props.run.Removed}
						sign="−"
						class="text-destructive-strong"
					/>
					<Count value={props.run.Listings} />
				</button>
				<Show when={open()}>
					<div class="flex flex-col gap-4 border-t border-border px-3 py-3 pl-10 text-data md:flex-row">
						<dl class="grid shrink-0 grid-cols-[5.5rem_1fr] gap-x-3 gap-y-1 self-start">
							<dt class="text-faint">Board</dt>
							<dd class="flex items-center gap-1.5">
								<SourceBadge source={props.run.Source} />
								<span class="font-mono text-xs text-faint">
									{props.token(props.run.BoardID)}
								</span>
							</dd>
							<dt class="text-faint">Elsewhere</dt>
							<dd>{props.run.Elsewhere} already found by a Search</dd>
							<dt class="text-faint">Trigger</dt>
							<dd class="capitalize">{props.run.Trigger}</dd>
							<Show when={props.run.FinishedAt}>
								{(finishedAt) => (
									<>
										<dt class="text-faint">Duration</dt>
										<dd>{formatDuration(props.run.StartedAt, finishedAt())}</dd>
									</>
								)}
							</Show>
						</dl>
						<Show when={props.run.Changes.length > 0}>
							<ul class="flex min-w-0 flex-1 flex-col gap-1">
								<For each={props.run.Changes}>
									{(change) => (
										<li class="flex items-center gap-2">
											<span
												class={cn(
													"w-16 shrink-0 text-xs font-medium",
													CHANGE_STYLE[change.Kind].class,
												)}
											>
												{CHANGE_STYLE[change.Kind].label}
											</span>
											<Link
												to="/jobs/$id"
												params={{ id: change.JobID }}
												class="truncate text-foreground hover:underline"
											>
												{change.Title}
											</Link>
										</li>
									)}
								</For>
							</ul>
						</Show>
						<Show when={props.run.Errors.length > 0}>
							<div class="flex min-w-0 flex-1 flex-col gap-2">
								<For each={props.run.Errors}>
									{(error) => (
										<p class="rounded-md border border-border bg-surface-muted p-3 font-mono text-xs text-destructive-strong">
											{error.Message}
										</p>
									)}
								</For>
							</div>
						</Show>
					</div>
				</Show>
			</div>
		</li>
	);
}
