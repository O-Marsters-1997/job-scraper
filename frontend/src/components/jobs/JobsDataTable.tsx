import {
	type ColumnDef,
	createSolidTable,
	flexRender,
	getCoreRowModel,
	type RowSelectionState,
} from "@tanstack/solid-table";
import { createEffect, createSignal, For, on, Show, untrack } from "solid-js";
import { Icon, type IconName } from "@/components/Icon";
import { JobFiltersDialog } from "@/components/jobs/JobFiltersDialog";
import { JobRowExpander } from "@/components/jobs/JobRowExpander";
import { Kbd } from "@/components/Kbd";
import { Pager } from "@/components/Pager";
import { ToggleChip } from "@/components/ToggleChip";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { useDebouncedTerm } from "@/hooks/useDebouncedTerm";
import { useShortcuts } from "@/hooks/useShortcuts";
import {
	activeFilterCount,
	type JobFilters,
	SORT_LABELS,
	SORT_OPTIONS,
} from "@/lib/jobFilters";
import { pageCount } from "@/lib/serverTable";
import type { Job } from "@/types/job";

export interface BulkAction<TData> {
	label: string;
	icon?: IconName;
	onSelect: (jobs: TData[]) => void;
}

interface JobsDataTableProps<TData extends Job> {
	columns: ColumnDef<TData, unknown>[];
	data: TData[];
	total: number;
	page: number;
	pageSize: number;
	onPage: (page: number) => void;
	filters: JobFilters;
	onChange: (patch: Partial<JobFilters>) => void;
	sourceOptions: { value: string; label: string }[];
	sortControl?: boolean;
	selection: RowSelectionState;
	onSelectionChange: (next: RowSelectionState) => void;
	selectedJobs: TData[];
	bulkActions?: BulkAction<TData>[];
	onMarkAllSeen?: (jobs: TData[]) => void;
	hidePosterFilter?: boolean;
}

const SORT_CHOICES = SORT_OPTIONS.map((value) => ({
	value,
	label: SORT_LABELS[value],
}));

const POSTER_CHIPS = [
	{ value: "direct", label: "Direct" },
	{ value: "agency", label: "Agency" },
	{ value: "all", label: "All" },
] as const;

export function JobsDataTable<TData extends Job>(
	props: JobsDataTableProps<TData>,
) {
	const [filtersOpen, setFiltersOpen] = createSignal(false);
	let selectionAnchor: string | undefined;
	const setSelection = (next: RowSelectionState) => {
		if (Object.keys(next).length === 0) selectionAnchor = undefined;
		props.onSelectionChange(next);
	};

	const [expandedRow, setExpandedRow] = createSignal<string | null>(null);
	const isExpanded = (rowId: string) => expandedRow() === rowId;
	const toggleExpanded = (rowId: string) =>
		setExpandedRow((prev) => (prev === rowId ? null : rowId));

	const lastPage = () => pageCount(props.total, props.pageSize);

	const selectRow = (rowId: string, extendRange: boolean) => {
		const ids = table.getRowModel().rows.map((row) => row.id);
		const from =
			selectionAnchor === undefined ? -1 : ids.indexOf(selectionAnchor);
		const to = ids.indexOf(rowId);
		const next = { ...props.selection };
		if (extendRange && from !== -1 && to !== -1) {
			for (const id of ids.slice(Math.min(from, to), Math.max(from, to) + 1)) {
				next[id] = true;
			}
		} else if (next[rowId]) {
			delete next[rowId];
		} else {
			next[rowId] = true;
		}
		setSelection(next);
		selectionAnchor = rowId;
	};

	const table = createSolidTable({
		get data() {
			return props.data;
		},
		get columns() {
			return props.columns;
		},
		getCoreRowModel: getCoreRowModel(),
		getRowId: (row) => row.ID,
		enableRowSelection: true,
		enableSorting: false,
		manualPagination: true,
		get pageCount() {
			return lastPage();
		},
		state: {
			get rowSelection() {
				return props.selection;
			},
			get pagination() {
				return { pageIndex: props.page - 1, pageSize: props.pageSize };
			},
		},
		onRowSelectionChange: (updater) =>
			setSelection(
				typeof updater === "function" ? updater(props.selection) : updater,
			),
		meta: {
			isExpanded,
			toggleExpanded,
			selectRow,
		},
	});

	let tableContainer: HTMLDivElement | undefined;
	let focusAfterPaging = false;

	const focusFirstRow = () => {
		tableContainer?.scrollIntoView({ block: "start" });
		tableContainer
			?.querySelector<HTMLElement>("tbody tr a")
			?.focus({ preventScroll: true });
	};

	createEffect(
		on(
			() => props.data,
			() => {
				if (!focusAfterPaging) return;
				focusAfterPaging = false;
				focusFirstRow();
			},
			{ defer: true },
		),
	);

	let searchInput: HTMLInputElement | undefined;
	const [searchFocused, setSearchFocused] = createSignal(false);
	const [searchTerm, search] = useDebouncedTerm();
	createEffect(
		on(searchTerm, (q) => props.onChange({ q }), {
			defer: true,
		}),
	);

	useShortcuts({
		"/": () => {
			searchInput?.scrollIntoView({ block: "nearest" });
			searchInput?.focus();
			searchInput?.select();
		},
		"]": () => {
			if (props.page >= lastPage()) return;
			focusAfterPaging = true;
			props.onPage(props.page + 1);
		},
		"[": () => {
			if (props.page <= 1) return;
			focusAfterPaging = true;
			props.onPage(props.page - 1);
		},
	});

	const unseenJobs = () => props.data.filter((job) => !job.Seen);
	const filterCount = () => activeFilterCount(props.filters);

	return (
		<div class="flex flex-col gap-3">
			<div class="flex min-h-9 flex-wrap items-center justify-between gap-2">
				<div class="flex min-w-0 flex-1 items-center gap-2">
					<div class="relative max-w-xs flex-1">
						<Icon
							name="search"
							size={14}
							class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-faint"
						/>
						<Label for="jobs-search" class="sr-only">
							Search jobs
						</Label>
						<Input
							ref={searchInput}
							id="jobs-search"
							aria-keyshortcuts="/"
							type="search"
							placeholder="Search by role or company…"
							value={untrack(() => props.filters.q)}
							onInput={(e) => search(e.currentTarget.value)}
							onFocus={() => setSearchFocused(true)}
							onBlur={() => setSearchFocused(false)}
							onKeyDown={(e) => {
								if (e.key !== "Enter" || e.isComposing) return;
								if (table.getRowModel().rows.length === 0) return;
								e.preventDefault();
								focusFirstRow();
							}}
							class="pr-3 pl-9"
						/>
						<Show when={!searchFocused() && props.filters.q === ""}>
							<Kbd class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2">
								/
							</Kbd>
						</Show>
					</div>
					<Button
						variant="outline"
						size="sm"
						onClick={() => setFiltersOpen(true)}
						class="relative shrink-0"
					>
						<Icon name="filter" size={14} />
						Filters
						<Show when={filterCount() > 0}>
							<Badge class="ml-0.5 h-4 min-w-4 justify-center px-1 py-0 text-2xs leading-none">
								{filterCount()}
							</Badge>
						</Show>
					</Button>
					<Show when={props.onMarkAllSeen && unseenJobs().length > 0}>
						<Button
							variant="outline"
							size="sm"
							class="shrink-0"
							onClick={() => props.onMarkAllSeen?.(unseenJobs())}
						>
							Mark {unseenJobs().length} as seen
						</Button>
					</Show>
				</div>
				<div class="ml-auto flex shrink-0 items-center gap-2">
					<Show when={!props.hidePosterFilter}>
						<For each={POSTER_CHIPS}>
							{(chip) => (
								<ToggleChip
									solid
									active={props.filters.poster === chip.value}
									onClick={() => props.onChange({ poster: chip.value })}
								>
									{chip.label}
								</ToggleChip>
							)}
						</For>
					</Show>
					<Show when={props.sortControl}>
						<Select
							options={SORT_CHOICES}
							optionValue={(o) => o?.value ?? ""}
							optionTextValue={(o) => o?.label ?? ""}
							disallowEmptySelection
							value={SORT_CHOICES.find((o) => o.value === props.filters.sort)}
							onChange={(opt) => {
								if (opt) props.onChange({ sort: opt.value });
							}}
							itemComponent={(itemProps) => (
								<SelectItem item={itemProps.item}>
									<SelectItemLabel>
										{itemProps.item.rawValue?.label}
									</SelectItemLabel>
								</SelectItem>
							)}
						>
							<SelectTrigger aria-label="Sort jobs">
								<Select.Value<(typeof SORT_CHOICES)[number]>>
									{(state) => state.selectedOption()?.label}
								</Select.Value>
							</SelectTrigger>
							<SelectContent />
						</Select>
					</Show>
				</div>
				<Show
					when={
						(props.bulkActions?.length ?? 0) > 0 &&
						props.selectedJobs.length > 0
					}
				>
					<div class="flex items-center gap-2">
						<span class="font-mono text-xs tabular-nums text-muted">
							{props.selectedJobs.length} selected
						</span>
						<Button variant="ghost" size="sm" onClick={() => setSelection({})}>
							Clear
						</Button>
						<DropdownMenu>
							<DropdownMenuTrigger as={Button} variant="outline" size="sm">
								Actions
								<Icon name="moreVertical" size={14} />
							</DropdownMenuTrigger>
							<DropdownMenuContent>
								<For each={props.bulkActions}>
									{(action) => (
										<DropdownMenuItem
											onSelect={() => action.onSelect(props.selectedJobs)}
										>
											<Show when={action.icon}>
												{(icon) => <Icon name={icon()} size={14} />}
											</Show>
											{action.label}
										</DropdownMenuItem>
									)}
								</For>
							</DropdownMenuContent>
						</DropdownMenu>
					</div>
				</Show>
			</div>

			<div
				ref={tableContainer}
				class="overflow-hidden rounded-xl border border-border bg-surface"
				data-feedback-collection={props.data.map((job) => job.ID).join(",")}
			>
				<Table class="[&_td]:px-2.5 [&_th]:px-2.5">
					<TableHeader>
						<For each={table.getHeaderGroups()}>
							{(headerGroup) => (
								<TableRow class="hover:bg-transparent">
									<For each={headerGroup.headers}>
										{(header) => (
											<TableHead>
												{header.isPlaceholder
													? null
													: flexRender(
															header.column.columnDef.header,
															header.getContext(),
														)}
											</TableHead>
										)}
									</For>
								</TableRow>
							)}
						</For>
					</TableHeader>
					<TableBody>
						<Show
							when={table.getRowModel().rows.length > 0}
							fallback={
								<tr>
									<td
										colspan={props.columns.length}
										class="py-12 text-center text-sm text-muted"
									>
										No jobs found.
									</td>
								</tr>
							}
						>
							<For each={table.getRowModel().rows}>
								{(row) => (
									<>
										<TableRow
											data-state={row.getIsSelected() ? "selected" : undefined}
											class={
												row.getIsSelected() ? "bg-accent-subtle/40" : undefined
											}
											data-feedback-job={
												row.original.SuitabilityScore == null
													? undefined
													: row.original.ID
											}
										>
											<For each={row.getVisibleCells()}>
												{(cell) => (
													<TableCell>
														{flexRender(
															cell.column.columnDef.cell,
															cell.getContext(),
														)}
													</TableCell>
												)}
											</For>
										</TableRow>
										<Show when={isExpanded(row.original.ID)}>
											<tr class="border-b border-border">
												<td colspan={row.getVisibleCells().length} class="p-0">
													<JobRowExpander job={row.original} />
												</td>
											</tr>
										</Show>
									</>
								)}
							</For>
						</Show>
					</TableBody>
				</Table>
				<Show when={props.total > 0}>
					<Pager
						total={props.total}
						page={props.page}
						pageSize={props.pageSize}
						noun="jobs"
						keyHints
						onPage={props.onPage}
					/>
				</Show>
			</div>

			<JobFiltersDialog
				open={filtersOpen()}
				onOpenChange={setFiltersOpen}
				filters={props.filters}
				onChange={(patch) =>
					props.onChange(
						props.hidePosterFilter
							? { ...patch, poster: props.filters.poster }
							: patch,
					)
				}
				sourceOptions={props.sourceOptions}
			/>
		</div>
	);
}
