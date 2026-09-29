import { createMemo, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { SortableTableHead } from "@/components/SortableTableHead";
import { SourceBadge } from "@/components/SourceBadge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	describeFilters,
	matchesSearch,
	paginate,
	type SearchParams,
	type SearchSortKey,
	type SearchStatus,
	sortTargets,
} from "@/lib/searchTargets";
import { cn } from "@/lib/utils";
import type { SourceInfo } from "../../../../types/source";
import type { SourceTarget } from "../../../../types/sourceTarget";
import {
	EnabledSwitch,
	FilterChips,
	Pager,
	RunStatus,
	SearchField,
} from "./parts";

const ALL = "all";

export function BoardSearchesTable(props: {
	targets: SourceTarget[];
	sources: SourceInfo[];
	params: SearchParams;
	onParams: (
		patch: Partial<SearchParams>,
		opts?: { replace?: boolean },
	) => void;
	onToggle: (target: SourceTarget) => void;
	onRun: (target: SourceTarget) => void;
	onDelete: (target: SourceTarget) => void;
	runPending: boolean;
}) {
	const info = (name: string) => props.sources.find((s) => s.name === name);
	const label = (name: string) => info(name)?.label ?? name;

	const filtered = createMemo(() =>
		sortTargets(
			props.targets.filter((t) =>
				matchesSearch(t, info(t.Source), props.params),
			),
			props.params.sort,
			props.params.dir,
		),
	);
	const paged = createMemo(() => paginate(filtered(), props.params.page));

	const filtersActive = () =>
		Boolean(props.params.q || props.params.src || props.params.status);

	const sourceOptions = () => [
		{ value: ALL, label: "All boards" },
		...props.sources.map((s) => ({
			value: s.name,
			label: s.label,
			count: props.targets.filter((t) => t.Source === s.name).length,
		})),
	];

	const head = (key: SearchSortKey, text: string, firstDir: "asc" | "desc") => {
		const sorted = () =>
			props.params.sort !== key
				? ("none" as const)
				: props.params.dir === "desc"
					? ("descending" as const)
					: ("ascending" as const);
		return (
			<SortableTableHead
				sorted={sorted()}
				onToggle={() =>
					props.onParams({
						sort: key,
						dir:
							props.params.sort === key
								? props.params.dir === "desc"
									? "asc"
									: "desc"
								: firstDir,
						page: undefined,
					})
				}
			>
				{text}
			</SortableTableHead>
		);
	};

	return (
		<div class="flex flex-col gap-3">
			<div class="flex flex-wrap items-center gap-2">
				<SearchField
					id="board-search"
					label="Search job board searches"
					placeholder="Search keywords or filters…"
					value={props.params.q ?? ""}
					onInput={(q) =>
						props.onParams(
							{ q: q || undefined, page: undefined },
							{ replace: true },
						)
					}
				/>
				<FilterChips
					label="Job board"
					value={props.params.src ?? ALL}
					onChange={(v) =>
						props.onParams({ src: v === ALL ? undefined : v, page: undefined })
					}
					options={sourceOptions()}
				/>
				<FilterChips
					label="Status"
					value={props.params.status ?? ALL}
					onChange={(v) =>
						props.onParams({
							status: v === ALL ? undefined : (v as SearchStatus),
							page: undefined,
						})
					}
					options={[
						{ value: ALL, label: "Any status" },
						{ value: "active", label: "Active" },
						{ value: "paused", label: "Paused" },
						{ value: "failed", label: "Failed" },
					]}
				/>
				<Show when={filtersActive()}>
					<button
						type="button"
						onClick={() =>
							props.onParams({
								q: undefined,
								src: undefined,
								status: undefined,
								page: undefined,
							})
						}
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
								{head("search", "Search", "asc")}
								<TableHead>Filters</TableHead>
								{head("lastRun", "Last run", "desc")}
								<TableHead class="w-24" />
								<TableHead class="w-20">Active</TableHead>
								<TableHead class="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							<For
								each={paged().items}
								fallback={
									<TableRow class="hover:bg-transparent">
										<TableCell
											colSpan={6}
											class="py-10 text-center text-sm text-muted"
										>
											{filtersActive()
												? "No searches match these filters."
												: "No searches yet. Use Build from fields to add one."}
										</TableCell>
									</TableRow>
								}
							>
								{(t) => {
									const busy = () =>
										t.RunStatus === "queued" || t.RunStatus === "running";
									const chips = () => describeFilters(t, info(t.Source));
									return (
										<TableRow
											class={cn("[&>td]:py-2.5", !t.Enabled && "text-faint")}
										>
											<TableCell>
												<div class="flex items-center gap-2">
													<SourceBadge source={label(t.Source)} />
													<span
														class="max-w-64 truncate font-medium"
														title={t.Value}
													>
														{t.Value}
													</span>
												</div>
											</TableCell>
											<TableCell>
												<div class="flex flex-wrap gap-1">
													<For
														each={chips()}
														fallback={
															<span class="text-xs text-faint">None</span>
														}
													>
														{(f) => (
															<span class="whitespace-nowrap rounded bg-surface-muted px-1.5 py-0.5 text-xs text-muted">
																{f.display}
															</span>
														)}
													</For>
												</div>
											</TableCell>
											<TableCell>
												<Show
													when={t.Enabled}
													fallback={<span class="text-xs">Paused</span>}
												>
													<RunStatus target={t} />
												</Show>
											</TableCell>
											<TableCell>
												<Button
													variant="outline"
													size="sm"
													class="w-full"
													onClick={() => props.onRun(t)}
													disabled={busy() || !t.Enabled || props.runPending}
												>
													{busy() ? "Running…" : "Run now"}
												</Button>
											</TableCell>
											<TableCell>
												<EnabledSwitch
													enabled={t.Enabled}
													name={t.Value}
													onToggle={() => props.onToggle(t)}
												/>
											</TableCell>
											<TableCell>
												<button
													type="button"
													onClick={() => props.onDelete(t)}
													aria-label={`Delete ${t.Value}`}
													title="Delete"
													class="grid size-7 place-items-center rounded text-faint transition hover:bg-destructive-subtle hover:text-destructive-strong focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-destructive"
												>
													<Icon name="trash" size={13} />
												</button>
											</TableCell>
										</TableRow>
									);
								}}
							</For>
						</TableBody>
					</Table>
				</div>
				<Pager
					from={paged().from}
					to={paged().to}
					total={paged().total}
					page={paged().page}
					pageCount={paged().pageCount}
					noun="searches"
					onPage={(page) =>
						props.onParams({ page: page > 1 ? page : undefined })
					}
				/>
			</Card>
		</div>
	);
}
