import { createMemo, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Pager } from "@/components/Pager";
import { SearchField } from "@/components/SearchField";
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
	SEARCH_PAGE_SIZE,
	type SearchParams,
	type SearchStatus,
	sortTargets,
} from "@/lib/searchTargets";
import { cn } from "@/lib/utils";
import type { SourceInfo } from "../../../../types/source";
import type { SourceTarget } from "../../../../types/sourceTarget";
import {
	DeleteIconButton,
	EmptyRow,
	EnabledSwitch,
	FilterChips,
	RunStatus,
	SortHead,
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
								<SortHead
									sortKey="search"
									firstDir="asc"
									params={props.params}
									onParams={props.onParams}
								>
									Search
								</SortHead>
								<TableHead>Filters</TableHead>
								<SortHead
									sortKey="lastRun"
									firstDir="desc"
									params={props.params}
									onParams={props.onParams}
								>
									Last run
								</SortHead>
								<TableHead class="w-24" />
								<TableHead class="w-20">Active</TableHead>
								<TableHead class="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							<For
								each={paged().items}
								fallback={
									<EmptyRow colSpan={6}>
										{filtersActive()
											? "No searches match these filters."
											: "No searches yet. Use Build from fields to add one."}
									</EmptyRow>
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
													<Show when={t.URL}>
														<a
															href={t.URL}
															target="_blank"
															rel="noopener noreferrer"
															aria-label={`Open on ${label(t.Source)}`}
															title={`Open on ${label(t.Source)}`}
															class="grid size-6 shrink-0 place-items-center rounded text-faint transition hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
														>
															<Icon name="externalLink" size={12} />
														</a>
													</Show>
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
												<DeleteIconButton
													label={`Delete ${t.Value}`}
													title="Delete"
													onClick={() => props.onDelete(t)}
												/>
											</TableCell>
										</TableRow>
									);
								}}
							</For>
						</TableBody>
					</Table>
				</div>
				<Pager
					total={paged().total}
					page={paged().page}
					pageSize={SEARCH_PAGE_SIZE}
					noun="searches"
					onPage={(page) =>
						props.onParams({ page: page > 1 ? page : undefined })
					}
				/>
			</Card>
		</div>
	);
}
