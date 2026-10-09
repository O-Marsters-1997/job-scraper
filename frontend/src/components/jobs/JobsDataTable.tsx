import {
	type ColumnDef,
	createSolidTable,
	flexRender,
	getCoreRowModel,
	getPaginationRowModel,
	getSortedRowModel,
	type RowSelectionState,
	type SortingState,
} from "@tanstack/solid-table";
import { createEffect, createSignal, For, on, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { JobFiltersDialog } from "@/components/jobs/JobFiltersDialog";
import { JobRowExpander } from "@/components/jobs/JobRowExpander";
import { Pager } from "@/components/Pager";
import { SortableTableHead } from "@/components/SortableTableHead";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Table,
	TableBody,
	TableCell,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useShortcuts } from "@/hooks/useShortcuts";
import {
	activeFilterCount,
	isDefaultView,
	type JobFilters,
} from "@/lib/jobFilters";
import type { Job } from "@/types/job";

interface JobsDataTableProps<TData extends Job> {
	columns: ColumnDef<TData, unknown>[];
	data: TData[];
	filters: JobFilters;
	onChange: (patch: Partial<JobFilters>) => void;
	page?: number | undefined;
	onPageChange?: (page: number | undefined) => void;
	sourceOptions: string[];
	wildcards?: boolean;
	selection?: RowSelectionState;
	onSelectionChange?: (next: RowSelectionState) => void;
	onBulkGrade?: (jobs: TData[]) => void;
	onBulkSeen?: (jobs: TData[], seen: boolean) => void;
	onMarkAllSeen?: (jobs: TData[]) => void;
}

const PAGE_SIZE = 10;

export function JobsDataTable<TData extends Job>(
	props: JobsDataTableProps<TData>,
) {
	const [sorting, setSorting] = createSignal<SortingState>([]);
	const [filtersOpen, setFiltersOpen] = createSignal(false);
	const [localSelection, setLocalSelection] = createSignal<RowSelectionState>(
		{},
	);
	const selection = () => props.selection ?? localSelection();
	const setSelection = (next: RowSelectionState) =>
		(props.onSelectionChange ?? setLocalSelection)(next);

	const [expandedRow, setExpandedRow] = createSignal<string | null>(null);
	const isExpanded = (rowId: string) => expandedRow() === rowId;
	const toggleExpanded = (rowId: string) =>
		setExpandedRow((prev) => (prev === rowId ? null : rowId));

	const [localPageIndex, setLocalPageIndex] = createSignal(0);
	const controlled = () => props.onPageChange !== undefined;
	const requestedPageIndex = () =>
		controlled() ? (props.page ?? 1) - 1 : localPageIndex();
	const pageIndex = () =>
		Math.min(
			requestedPageIndex(),
			Math.max(0, Math.ceil(props.data.length / PAGE_SIZE) - 1),
		);
	const setPageIndex = (next: number) => {
		if (props.onPageChange) props.onPageChange(next > 0 ? next + 1 : undefined);
		else setLocalPageIndex(next);
	};

	createEffect(
		on(
			() => props.filters,
			() => setLocalPageIndex(0),
			{ defer: true },
		),
	);

	createEffect(() => {
		if (!controlled() || props.data.length === 0) return;
		if (requestedPageIndex() > pageIndex()) setPageIndex(pageIndex());
	});

	const table = createSolidTable({
		get data() {
			return props.data;
		},
		get columns() {
			return props.columns;
		},
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		getRowId: (row) => row.ID,
		enableRowSelection: true,
		autoResetPageIndex: false,
		state: {
			get sorting() {
				return sorting();
			},
			get rowSelection() {
				return selection();
			},
			get pagination() {
				return { pageIndex: pageIndex(), pageSize: PAGE_SIZE };
			},
		},
		onSortingChange: (updater) => {
			setSorting(updater);
			setPageIndex(0);
		},
		onPaginationChange: (updater) => {
			const next =
				typeof updater === "function"
					? updater({ pageIndex: pageIndex(), pageSize: PAGE_SIZE })
					: updater;
			setPageIndex(next.pageIndex);
		},
		onRowSelectionChange: (updater) =>
			setSelection(
				typeof updater === "function" ? updater(selection()) : updater,
			),
		meta: {
			isExpanded,
			toggleExpanded,
			showWildcard: () =>
				props.wildcards === true &&
				isDefaultView(props.filters, sorting().length > 0),
		},
	});

	createEffect(
		on(
			[() => props.data, sorting, () => table.getState().pagination.pageIndex],
			() => setSelection({}),
			{ defer: true },
		),
	);

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
			() => table.getState().pagination.pageIndex,
			() => {
				if (!focusAfterPaging) return;
				focusAfterPaging = false;
				focusFirstRow();
			},
			{ defer: true },
		),
	);

	useShortcuts({
		"]": () => {
			if (!table.getCanNextPage()) return;
			focusAfterPaging = true;
			table.nextPage();
		},
		"[": () => {
			if (!table.getCanPreviousPage()) return;
			focusAfterPaging = true;
			table.previousPage();
		},
	});

	const selectedJobs = () =>
		props.data.filter((job) => selection()[job.ID] === true);
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
							id="jobs-search"
							type="search"
							placeholder="Search by role or company…"
							value={props.filters.q}
							onInput={(e) => props.onChange({ q: e.currentTarget.value })}
							class="pr-3 pl-9"
						/>
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
							<Badge class="ml-0.5 h-4 min-w-4 px-1 text-[10px]">
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
				<Show when={props.onBulkGrade && selectedJobs().length > 0}>
					<div class="flex items-center gap-2">
						<span class="font-mono text-xs tabular-nums text-muted">
							{selectedJobs().length} selected
						</span>
						<Button variant="ghost" size="sm" onClick={() => setSelection({})}>
							Clear
						</Button>
						<Button
							size="sm"
							onClick={() => props.onBulkGrade?.(selectedJobs())}
						>
							<Icon name="check" size={14} />
							Grade
						</Button>
						<Show when={props.onBulkSeen}>
							<Button
								variant="outline"
								size="sm"
								onClick={() => props.onBulkSeen?.(selectedJobs(), true)}
							>
								Mark seen
							</Button>
							<Button
								variant="outline"
								size="sm"
								onClick={() => props.onBulkSeen?.(selectedJobs(), false)}
							>
								Mark unseen
							</Button>
						</Show>
					</div>
				</Show>
			</div>

			<div
				ref={tableContainer}
				class="overflow-hidden rounded-xl border border-border bg-surface"
				data-feedback-collection={table
					.getPrePaginationRowModel()
					.rows.map((row) => row.original.ID)
					.join(",")}
			>
				<Table class="[&_td]:px-2.5 [&_th]:px-2.5">
					<TableHeader>
						<For each={table.getHeaderGroups()}>
							{(headerGroup) => (
								<TableRow class="hover:bg-transparent">
									<For each={headerGroup.headers}>
										{(header) => (
											<SortableTableHead
												sorted={
													header.column.getCanSort()
														? header.column.getIsSorted() || "none"
														: false
												}
												onToggle={(e) =>
													header.column.getToggleSortingHandler()?.(e)
												}
											>
												{header.isPlaceholder
													? null
													: flexRender(
															header.column.columnDef.header,
															header.getContext(),
														)}
											</SortableTableHead>
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
				<Show when={props.data.length > 0}>
					<Pager
						total={props.data.length}
						page={pageIndex() + 1}
						pageSize={PAGE_SIZE}
						noun="jobs"
						keyHints
						onPage={(page) => table.setPageIndex(page - 1)}
					/>
				</Show>
			</div>

			<JobFiltersDialog
				open={filtersOpen()}
				onOpenChange={setFiltersOpen}
				filters={props.filters}
				onChange={props.onChange}
				sourceOptions={props.sourceOptions}
			/>
		</div>
	);
}
