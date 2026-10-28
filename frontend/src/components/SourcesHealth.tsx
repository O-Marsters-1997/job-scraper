import { formatDistanceToNowStrict } from "date-fns";
import { createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { SourceBadge } from "@/components/SourceBadge";
import { Button } from "@/components/ui/button";
import { byNew24h, healthLabel, splitHealth } from "@/lib/scrapeRuns";
import { searchTitle } from "@/lib/searchTargets";
import { cn } from "@/lib/utils";
import type { ScrapeRunOutcome, SourceHealth } from "@/types/scrapeRun";

const STATUS_DOT = {
	red: "bg-destructive",
	amber: "bg-status-interview",
	green: "bg-status-offer",
} as const;

const STATUS_TEXT = {
	red: "text-destructive-strong",
	amber: "text-foreground",
	green: "text-muted",
} as const;

const OUTCOME_BAR: Record<ScrapeRunOutcome, string> = {
	running: "bg-primary",
	succeeded: "bg-status-offer",
	retried: "bg-status-interview",
	failed: "bg-destructive",
};

const GRID =
	"grid grid-cols-[minmax(0,1fr)_11rem_5rem_8rem_3.5rem_5rem_5.5rem_4rem] items-center gap-x-3";

function Total(props: { value: number; class?: string }) {
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

function RunStrip(props: { recent: ScrapeRunOutcome[] }) {
	const empty = () => Math.max(0, 5 - props.recent.length);
	return (
		<span
			role="img"
			aria-label={`Last runs, oldest first: ${props.recent.join(", ") || "none"}`}
			class="flex gap-0.5"
		>
			<For each={Array.from({ length: empty() })}>
				{() => <span class="h-4 w-1.5 rounded-sm bg-border" />}
			</For>
			<For each={props.recent}>
				{(outcome) => (
					<span class={cn("h-4 w-1.5 rounded-sm", OUTCOME_BAR[outcome])} />
				)}
			</For>
		</span>
	);
}

function HealthRow(props: {
	row: SourceHealth;
	sourceLabel: string;
	selected: boolean;
	onSelect: () => void;
}) {
	const title = () =>
		props.row.Kind === "board"
			? props.row.CompanyName
			: searchTitle({
					Source: props.row.Source,
					Value: props.row.TargetValue,
				});
	return (
		<li class="border-b border-border last:border-b-0">
			<button
				type="button"
				aria-pressed={props.selected}
				onClick={() => props.onSelect()}
				class={cn(
					GRID,
					"w-full px-4 py-2.5 text-left text-data transition-colors hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary",
					props.selected && "bg-surface-muted",
				)}
			>
				<span class="flex min-w-0 items-center gap-2">
					<SourceBadge source={props.sourceLabel} />
					<span class="truncate text-foreground">{title()}</span>
				</span>
				<span
					class={cn("flex items-center gap-2", STATUS_TEXT[props.row.Status])}
				>
					<span
						class={cn("size-2 rounded-full", STATUS_DOT[props.row.Status])}
					/>
					{healthLabel(props.row)}
				</span>
				<RunStrip recent={props.row.Recent} />
				<span class="text-muted">
					<Show when={props.row.LastRunAt} fallback="—">
						{(at) => `${formatDistanceToNowStrict(new Date(at()))} ago`}
					</Show>
				</span>
				<Total value={props.row.New} class="font-semibold" />
				<Total value={props.row.New24h} class="font-semibold" />
				<Total value={props.row.Elsewhere} />
				<Total
					value={props.row.Retries}
					class={props.row.Retries > 0 ? "text-destructive-strong" : ""}
				/>
			</button>
		</li>
	);
}

export function SourcesHealth(props: {
	rows: SourceHealth[];
	selected: string | undefined;
	sourceLabel: (name: string) => string;
	onSelect: (targetId: string) => void;
}) {
	const [showHealthy, setShowHealthy] = createSignal(false);
	const [onlyNew, setOnlyNew] = createSignal(false);
	const [sortByNew, setSortByNew] = createSignal(false);
	const groups = () => splitHealth(props.rows);
	const flat = () =>
		onlyNew() || sortByNew()
			? byNew24h(props.rows, { onlyNew: onlyNew(), sortByNew: sortByNew() })
			: undefined;
	const renderRow = (row: SourceHealth) => (
		<HealthRow
			row={row}
			sourceLabel={props.sourceLabel(row.Source)}
			selected={row.Kind === "search" && props.selected === row.TargetID}
			onSelect={() => row.Kind === "search" && props.onSelect(row.TargetID)}
		/>
	);
	return (
		<section class="mb-8" aria-labelledby="sources-heading">
			<h2 id="sources-heading" class="mb-3 text-base font-semibold">
				Sources
			</h2>
			<Show
				when={props.rows.length > 0}
				fallback={
					<p class="rounded-xl border border-border bg-surface p-6 text-center text-sm text-muted">
						No searches yet.
					</p>
				}
			>
				<div class="mb-2 flex gap-2">
					<Button
						variant={onlyNew() ? "secondary" : "outline"}
						size="sm"
						aria-pressed={onlyNew()}
						onClick={() => setOnlyNew((v) => !v)}
					>
						Has new (24h)
					</Button>
					<Button
						variant={sortByNew() ? "secondary" : "outline"}
						size="sm"
						aria-pressed={sortByNew()}
						onClick={() => setSortByNew((v) => !v)}
					>
						Sort by New (24h)
					</Button>
				</div>
				<div class="overflow-hidden rounded-xl border border-border bg-surface">
					<div
						class={cn(
							GRID,
							"border-b border-border bg-surface-muted px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-faint",
						)}
					>
						<span>Search</span>
						<span>Status</span>
						<span>Last 5</span>
						<span>Last run</span>
						<span class="text-right">New</span>
						<span class="text-right">New 24h</span>
						<span class="text-right">Elsewhere</span>
						<span class="text-right">Retries</span>
					</div>
					<ul>
						<Show
							when={flat()}
							fallback={
								<>
									<For each={groups().attention}>{renderRow}</For>
									<Show when={showHealthy()}>
										<For each={groups().healthy}>{renderRow}</For>
									</Show>
								</>
							}
						>
							{(rows) => (
								<For
									each={rows()}
									fallback={
										<li class="px-4 py-3 text-sm text-muted">
											No sources with new jobs in the last 24 hours.
										</li>
									}
								>
									{renderRow}
								</For>
							)}
						</Show>
					</ul>
					<Show when={!flat() && groups().healthy.length > 0}>
						<div class="flex justify-center border-t border-border py-1.5">
							<Button
								variant="ghost"
								size="sm"
								aria-expanded={showHealthy()}
								onClick={() => setShowHealthy((v) => !v)}
							>
								{showHealthy()
									? "Hide healthy sources"
									: `Show ${groups().healthy.length} healthy ${groups().healthy.length === 1 ? "source" : "sources"}`}
								<Icon
									name={showHealthy() ? "chevronUp" : "chevronDown"}
									class="size-3.5"
								/>
							</Button>
						</div>
					</Show>
				</div>
				<p class="mt-2 text-xs text-muted">
					Totals cover each source's last 5 runs. New 24h counts jobs first
					found in the last 24 hours that pass your filters. Select a row to
					filter the run log.
				</p>
			</Show>
		</section>
	);
}
