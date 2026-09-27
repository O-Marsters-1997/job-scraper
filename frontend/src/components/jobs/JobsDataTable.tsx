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
import { JobFiltersDialog } from "@/components/jobs/JobFiltersDialog";
import { JobRowExpander } from "@/components/jobs/JobRowExpander";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { activeFilterCount, type JobFilters } from "@/lib/jobFilters";
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
		},
	});

	const filterCount = () => activeFilterCount(props.filters);

	return (
		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-2">
				<div class="relative max-w-xs flex-1">
					<svg
						aria-hidden="true"
						class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-faint"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<circle cx="11" cy="11" r="8" />
						<line x1="21" y1="21" x2="16.65" y2="16.65" />
					</svg>
					<Input
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
					<svg
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<line x1="4" y1="6" x2="20" y2="6" />
						<line x1="8" y1="12" x2="16" y2="12" />
						<line x1="11" y1="18" x2="13" y2="18" />
					</svg>
					Filters
					<Show when={filterCount() > 0}>
						<Badge class="ml-0.5 h-4 min-w-4 px-1 text-[10px]">
							{filterCount()}
						</Badge>
					</Show>
				</Button>
			</div>

			<div class="overflow-hidden rounded-xl border border-border bg-surface">
				<Table>
					<TableHeader>
						<For each={table.getHeaderGroups()}>
							{(headerGroup) => (
								<TableRow class="hover:bg-transparent">
									<For each={headerGroup.headers}>
										{(header) => (
											<TableHead
												class={
													header.column.getCanSort()
														? "cursor-pointer select-none"
														: undefined
												}
												onClick={header.column.getToggleSortingHandler()}
											>
												<div class="flex items-center gap-1">
													{header.isPlaceholder
														? null
														: flexRender(
																header.column.columnDef.header,
																header.getContext(),
															)}
													<Show when={header.column.getCanSort()}>
														<span class="text-faint">
															{header.column.getIsSorted() === "asc"
																? "↑"
																: header.column.getIsSorted() === "desc"
																	? "↓"
																	: "↕"}
														</span>
													</Show>
												</div>
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
										<TableRow>
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
