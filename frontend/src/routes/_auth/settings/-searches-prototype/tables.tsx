import { Link } from "@tanstack/solid-router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { SortableTableHead } from "@/components/SortableTableHead";
import { SourceBadge } from "@/components/SourceBadge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import type { SourceTarget } from "@/types/sourceTarget";
import {
	BOARD_SPECS,
	buildBoardUrl,
	describeFilters,
	draftFromTarget,
} from "./boardUrl";
import { EnabledSwitch, RunStatus } from "./parts";
import {
	ariaSort,
	FilterChips,
	nextSort,
	Pager,
	SearchField,
	type Sort,
	usePaged,
} from "./tableControls";
import {
	type CompanyBoardRow,
	type ProtoSearches,
	relativeTime,
} from "./useProtoSearches";

const COMPANY_PAGE_SIZE = 25;
const SEARCH_PAGE_SIZE = 10;

function EmptyRow(props: { cols: number; children: string }) {
	return (
		<TableRow class="hover:bg-transparent">
			<TableCell
				colSpan={props.cols}
				class="py-10 text-center text-sm text-muted"
			>
				{props.children}
			</TableCell>
		</TableRow>
	);
}

function DeleteButton(props: { name: string; onClick: () => void }) {
	return (
		<button
			type="button"
			onClick={props.onClick}
			aria-label={`Delete ${props.name}`}
			title="Delete"
			class="grid size-7 place-items-center rounded text-faint transition hover:bg-destructive-subtle hover:text-destructive-strong focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-destructive"
		>
			<Icon name="trash" size={13} />
		</button>
	);
}

type TrackingFilter = "all" | "tracking" | "paused";
type CompanySortKey = "name" | "relevant" | "checked";
type AtsOption = { value: string; label: string };
const ALL_ATS = "__all__";

export function CompanyBoardsTable(props: { s: ProtoSearches }) {
	const [q, setQ] = createSignal("");
	const [tracking, setTracking] = createSignal<TrackingFilter>("all");
	const [ats, setAts] = createSignal(ALL_ATS);
	const [onlyRelevant, setOnlyRelevant] = createSignal(false);
	const [sort, setSort] = createSignal<Sort<CompanySortKey>>({
		key: "relevant",
		dir: "desc",
	});

	const rows = createMemo(() => props.s.companyRows());

	const atsOptions = (): AtsOption[] => [
		{ value: ALL_ATS, label: "All ATS" },
		...[...new Set(rows().map((r) => r.target.Source))]
			.sort()
			.map((value) => ({ value, label: props.s.label(value) })),
	];

	const filtered = createMemo(() => {
		const needle = q().trim().toLowerCase();
		const list = rows().filter(
			(r) =>
				(!needle ||
					r.name.toLowerCase().includes(needle) ||
					r.target.Value.includes(needle)) &&
				(ats() === ALL_ATS || r.target.Source === ats()) &&
				(tracking() === "all" ||
					(tracking() === "tracking") === r.target.Enabled) &&
				(!onlyRelevant() || r.relevant > 0),
		);
		const { key, dir } = sort();
		const by: Record<
			CompanySortKey,
			(a: CompanyBoardRow, b: CompanyBoardRow) => number
		> = {
			name: (a, b) => a.name.localeCompare(b.name),
			relevant: (a, b) => a.relevant - b.relevant || a.open - b.open,
			checked: (a, b) =>
				new Date(a.target.LastRunAt ?? 0).getTime() -
				new Date(b.target.LastRunAt ?? 0).getTime(),
		};
		const sign = dir === "asc" ? 1 : -1;
		return [...list].sort((a, b) => sign * by[key](a, b));
	});

	const paged = usePaged(filtered, COMPANY_PAGE_SIZE, () => [
		q(),
		tracking(),
		ats(),
		onlyRelevant(),
		sort(),
	]);

	const counts = () => ({
		all: rows().length,
		tracking: rows().filter((r) => r.target.Enabled).length,
		paused: rows().filter((r) => !r.target.Enabled).length,
	});

	const filtersActive = () =>
		q() || tracking() !== "all" || ats() !== ALL_ATS || onlyRelevant();

	const clearFilters = () => {
		setQ("");
		setTracking("all");
		setAts(ALL_ATS);
		setOnlyRelevant(false);
	};

	const head = (
		key: CompanySortKey,
		label: string,
		firstDir?: "asc" | "desc",
	) => (
		<SortableTableHead
			sorted={ariaSort(sort(), key)}
			onToggle={() => setSort(nextSort(sort(), key, firstDir))}
		>
			<span class="uppercase tracking-wide">{label}</span>
		</SortableTableHead>
	);

	return (
		<div class="flex flex-col gap-3">
			<div class="flex flex-wrap items-center gap-2">
				<SearchField
					id="company-search"
					label="Search company boards"
					placeholder="Search companies…"
					value={q()}
					onInput={setQ}
				/>
				<Select<AtsOption>
					options={atsOptions()}
					optionValue="value"
					optionTextValue="label"
					value={atsOptions().find((o) => o.value === ats()) ?? null}
					onChange={(o) => setAts(o?.value ?? ALL_ATS)}
					itemComponent={(p) => (
						<SelectItem item={p.item}>
							<SelectItemLabel>{p.item.rawValue.label}</SelectItemLabel>
						</SelectItem>
					)}
				>
					<Select.Label class="sr-only">Filter by ATS</Select.Label>
					<SelectTrigger class="h-8 w-36 text-xs">
						<Select.Value<AtsOption>>
							{(state) => state.selectedOption()?.label ?? "All ATS"}
						</Select.Value>
					</SelectTrigger>
					<SelectContent />
				</Select>
				<FilterChips<TrackingFilter>
					label="Tracking"
					value={tracking()}
					onChange={setTracking}
					options={[
						{ value: "all", label: "All", count: counts().all },
						{ value: "tracking", label: "Tracking", count: counts().tracking },
						{ value: "paused", label: "Paused", count: counts().paused },
					]}
				/>
				<button
					type="button"
					aria-pressed={onlyRelevant()}
					onClick={() => setOnlyRelevant(!onlyRelevant())}
					class={cn(
						"inline-flex h-7 items-center gap-1.5 rounded-full border px-2.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
						onlyRelevant()
							? "border-accent-border bg-accent-subtle text-accent-text"
							: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
					)}
				>
					<Show when={onlyRelevant()}>
						<Icon name="check" size={11} strokeWidth={2.5} />
					</Show>
					Has relevant roles
				</button>
				<Show when={filtersActive()}>
					<button
						type="button"
						onClick={clearFilters}
						class="text-xs font-medium text-primary hover:underline"
					>
						Clear
					</button>
				</Show>
			</div>

			<Card class="overflow-hidden">
				<div class="overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow class="hover:bg-transparent">
								{head("name", "Company")}
								{head("relevant", "Relevant roles", "desc")}
								{head("checked", "Last checked", "desc")}
								<TableHead class="w-20">Tracking</TableHead>
								<TableHead class="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							<For
								each={paged.pageItems()}
								fallback={
									<EmptyRow cols={5}>
										{filtersActive()
											? "No company boards match these filters."
											: "Paste a company's careers page above to track it."}
									</EmptyRow>
								}
							>
								{(r) => <CompanyRow s={props.s} r={r} />}
							</For>
						</TableBody>
					</Table>
				</div>
				<Pager paged={paged} noun="company boards" />
			</Card>
		</div>
	);
}

function CompanyRow(props: { s: ProtoSearches; r: CompanyBoardRow }) {
	const t = () => props.r.target;
	return (
		<TableRow class={cn("group/row [&>td]:py-2", !t().Enabled && "text-faint")}>
			<TableCell>
				<div class="flex items-center gap-2.5">
					<a
						href={props.r.boardUrl}
						target="_blank"
						rel="noreferrer"
						title={`Open ${t().Value} on ${props.s.label(t().Source)}`}
						class="group inline-flex w-28 shrink-0 items-center rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
					>
						<SourceBadge
							source={props.s.label(t().Source)}
							class="transition-colors group-hover:bg-accent-subtle group-hover:text-accent-text"
						/>
						<Icon
							name="link"
							size={10}
							class="ml-1 text-faint opacity-0 transition group-hover:text-accent-text group-hover/row:opacity-100 group-focus-visible:opacity-100"
						/>
						<span class="sr-only">(opens in a new tab)</span>
					</a>
					<Show
						when={props.r.companyId}
						fallback={<span class="font-medium">{props.r.name}</span>}
					>
						{(id) => (
							<Link
								to="/companies/$id"
								params={{ id: id() }}
								class="truncate font-medium underline-offset-2 hover:text-primary hover:underline"
							>
								{props.r.name}
							</Link>
						)}
					</Show>
				</div>
			</TableCell>
			<TableCell>
				<span
					class="inline-flex items-baseline gap-1 whitespace-nowrap text-xs"
					title="Open roles on this board that pass your search config's relevance cutoff"
				>
					<Show
						when={props.r.relevant > 0}
						fallback={
							<span class="font-mono text-sm tabular-nums text-faint">0</span>
						}
					>
						<Link
							to="/jobs"
							search={{ q: props.r.name }}
							class="font-mono text-sm font-medium tabular-nums text-foreground underline-offset-2 hover:text-primary hover:underline"
						>
							{props.r.relevant}
						</Link>
					</Show>
					<span class="text-faint">of {props.r.open} open</span>
				</span>
			</TableCell>
			<TableCell class="whitespace-nowrap text-xs text-muted">
				{t().Enabled ? relativeTime(t().LastRunAt) : "Paused"}
			</TableCell>
			<TableCell>
				<EnabledSwitch
					target={t()}
					name={props.r.name}
					onToggle={() => props.s.toggle(t())}
				/>
			</TableCell>
			<TableCell>
				<DeleteButton
					name={props.r.name}
					onClick={() => props.s.remove(t().ID)}
				/>
			</TableCell>
		</TableRow>
	);
}

type SearchStatusFilter = "all" | "active" | "paused" | "failed";

export function BoardSearchesTable(props: { s: ProtoSearches }) {
	const [q, setQ] = createSignal("");
	const [source, setSource] = createSignal("all");
	const [status, setStatus] = createSignal<SearchStatusFilter>("all");

	const items = () => props.s.list("boards");
	const filtered = createMemo(() => {
		const needle = q().trim().toLowerCase();
		return items().filter((t) => {
			const d = draftFromTarget(t);
			const haystack = [d.keywords, ...describeFilters(d).map((f) => f.display)]
				.join(" ")
				.toLowerCase();
			return (
				(!needle || haystack.includes(needle)) &&
				(source() === "all" || t.Source === source()) &&
				(status() === "all" ||
					(status() === "active" && t.Enabled) ||
					(status() === "paused" && !t.Enabled) ||
					(status() === "failed" && t.RunStatus === "failed"))
			);
		});
	});
	const paged = usePaged(filtered, SEARCH_PAGE_SIZE, () => [
		q(),
		source(),
		status(),
	]);

	const filtersActive = () => q() || source() !== "all" || status() !== "all";

	return (
		<div class="flex flex-col gap-3">
			<div class="flex flex-wrap items-center gap-2">
				<SearchField
					id="board-search"
					label="Search job board searches"
					placeholder="Search keywords or filters…"
					value={q()}
					onInput={setQ}
				/>
				<FilterChips
					label="Job board"
					value={source()}
					onChange={setSource}
					options={[
						{ value: "all", label: "All boards" },
						...BOARD_SPECS.map((b) => ({
							value: b.source,
							label: b.label,
							count: items().filter((t) => t.Source === b.source).length,
						})),
					]}
				/>
				<FilterChips<SearchStatusFilter>
					label="Status"
					value={status()}
					onChange={setStatus}
					options={[
						{ value: "all", label: "Any status" },
						{ value: "active", label: "Active" },
						{ value: "paused", label: "Paused" },
						{ value: "failed", label: "Failed" },
					]}
				/>
				<Show when={filtersActive()}>
					<button
						type="button"
						onClick={() => {
							setQ("");
							setSource("all");
							setStatus("all");
						}}
						class="text-xs font-medium text-primary hover:underline"
					>
						Clear
					</button>
				</Show>
			</div>

			<Card class="overflow-hidden">
				<div class="overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow class="hover:bg-transparent">
								<TableHead>Search</TableHead>
								<TableHead>Filters</TableHead>
								<TableHead>Last run</TableHead>
								<TableHead class="w-24" />
								<TableHead class="w-20">Active</TableHead>
								<TableHead class="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							<For
								each={paged.pageItems()}
								fallback={
									<EmptyRow cols={6}>
										{filtersActive()
											? "No searches match these filters."
											: "Paste a LinkedIn, Indeed or Work in Startups search above."}
									</EmptyRow>
								}
							>
								{(t) => <SearchRow s={props.s} t={t} />}
							</For>
						</TableBody>
					</Table>
				</div>
				<Pager paged={paged} noun="searches" />
			</Card>
		</div>
	);
}

function SearchRow(props: { s: ProtoSearches; t: SourceTarget }) {
	const d = () => draftFromTarget(props.t);
	const busy = () =>
		props.t.RunStatus === "queued" || props.t.RunStatus === "running";
	const title = () => d().keywords || "Untitled search";
	return (
		<TableRow class={cn("[&>td]:py-2.5", !props.t.Enabled && "text-faint")}>
			<TableCell>
				<div class="flex items-center gap-2">
					<SourceBadge source={props.s.label(props.t.Source)} />
					<a
						href={buildBoardUrl(d())}
						target="_blank"
						rel="noreferrer"
						title="Open page 1 on the job board"
						class="max-w-64 truncate font-medium underline-offset-2 hover:text-primary hover:underline"
					>
						{title()}
					</a>
				</div>
			</TableCell>
			<TableCell>
				<div class="flex flex-wrap gap-1">
					<For
						each={describeFilters(d())}
						fallback={<span class="text-xs text-faint">None</span>}
					>
						{(f) => (
							<span
								class="whitespace-nowrap rounded bg-surface-muted px-1.5 py-0.5 text-xs text-muted"
								title={`${f.label}: ${f.key}=${f.value}`}
							>
								{f.display}
							</span>
						)}
					</For>
				</div>
			</TableCell>
			<TableCell>
				<Show
					when={props.t.Enabled}
					fallback={<span class="text-xs">Paused</span>}
				>
					<RunStatus target={props.t} />
				</Show>
			</TableCell>
			<TableCell>
				<Button
					variant="outline"
					size="sm"
					class="w-full"
					onClick={() => props.s.run(props.t.ID)}
					disabled={busy() || !props.t.Enabled}
				>
					{busy() ? "Running…" : "Run now"}
				</Button>
			</TableCell>
			<TableCell>
				<EnabledSwitch
					target={props.t}
					name={title()}
					onToggle={() => props.s.toggle(props.t)}
				/>
			</TableCell>
			<TableCell>
				<DeleteButton
					name={title()}
					onClick={() => props.s.remove(props.t.ID)}
				/>
			</TableCell>
		</TableRow>
	);
}
