import {
	type ColumnDef,
	createSolidTable,
	flexRender,
	getCoreRowModel,
	getPaginationRowModel,
	getSortedRowModel,
	type SortingState,
} from "@tanstack/solid-table";
import { createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { JobFiltersDialog } from "@/components/jobs/JobFiltersDialog";
import { JobRowExpander } from "@/components/jobs/JobRowExpander";
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
	sourceOptions: string[];
}

export function JobsDataTable<TData extends Job>(
	props: JobsDataTableProps<TData>,
) {
	const [sorting, setSorting] = createSignal<SortingState>([]);
	const [filtersOpen, setFiltersOpen] = createSignal(false);

	const [expandedRow, setExpandedRow] = createSignal<string | null>(null);
	const isExpanded = (rowId: string) => expandedRow() === rowId;
	const toggleExpanded = (rowId: string) =>
		setExpandedRow((prev) => (prev === rowId ? null : rowId));

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
		initialState: { pagination: { pageSize: 10 } },
		state: {
			get sorting() {
				return sorting();
			},
		},
		onSortingChange: setSorting,
		meta: {
			isExpanded,
			toggleExpanded,
			showWildcard: () => isDefaultView(props.filters, sorting().length > 0),
		},
	});

	const filterCount = () => activeFilterCount(props.filters);

	return (
		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-2">
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
			</div>

			<div
				class="overflow-hidden rounded-xl border border-border bg-surface"
				data-feedback-collection={table
					.getPrePaginationRowModel()
					.rows.map((row) => row.original.ID)
					.join(",")}
			>
				<Table>
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
				<Show when={table.getPageCount() > 1}>
					<div class="flex items-center justify-end gap-2 border-t border-border px-4 py-2.5">
						<span class="mr-auto text-xs text-faint">
							Page {table.getState().pagination.pageIndex + 1} of{" "}
							{table.getPageCount()}
						</span>
						<Button
							variant="outline"
							size="sm"
							disabled={!table.getCanPreviousPage()}
							onClick={() => table.previousPage()}
						>
							Previous
						</Button>
						<Button
							variant="outline"
							size="sm"
							disabled={!table.getCanNextPage()}
							onClick={() => table.nextPage()}
						>
							Next
						</Button>
					</div>
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
