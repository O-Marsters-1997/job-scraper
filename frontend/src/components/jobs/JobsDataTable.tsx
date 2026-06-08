import { For, Show } from "solid-js";
import { createSignal } from "solid-js";
import {
	createSolidTable,
	flexRender,
	getCoreRowModel,
	getFilteredRowModel,
	getPaginationRowModel,
	getSortedRowModel,
	type ColumnDef,
	type PaginationState,
	type SortingState,
} from "@tanstack/solid-table";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";

interface JobsDataTableProps<TData> {
	columns: ColumnDef<TData, unknown>[];
	data: TData[];
}

export function JobsDataTable<TData>(props: JobsDataTableProps<TData>) {
	const [globalFilter, setGlobalFilter] = createSignal("");
	const [sorting, setSorting] = createSignal<SortingState>([]);
	const [pagination, setPagination] = createSignal<PaginationState>({
		pageIndex: 0,
		pageSize: 20,
	});

	const table = createSolidTable({
		get data() {
			return props.data;
		},
		columns: props.columns,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		globalFilterFn: "includesString",
		state: {
			get globalFilter() {
				return globalFilter();
			},
			get sorting() {
				return sorting();
			},
			get pagination() {
				return pagination();
			},
		},
		onGlobalFilterChange: setGlobalFilter,
		onSortingChange: setSorting,
		onPaginationChange: setPagination,
	});

	const pageIndex = () => table.getState().pagination.pageIndex;
	const pageCount = () => table.getPageCount();
	const filteredCount = () => table.getFilteredRowModel().rows.length;
	const start = () => pageIndex() * pagination().pageSize + 1;
	const end = () =>
		Math.min((pageIndex() + 1) * pagination().pageSize, filteredCount());

	return (
		<div class="space-y-3">
			{/* Search */}
			<div class="relative max-w-xs">
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
				<input
					type="search"
					placeholder="Search by role or company…"
					value={globalFilter()}
					onInput={(e) => {
						setGlobalFilter(e.currentTarget.value);
						setPagination((p) => ({ ...p, pageIndex: 0 }));
					}}
					class="w-full rounded-md border border-border bg-surface py-2 pr-3 pl-9 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
				/>
			</div>

			{/* Table */}
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
								)}
							</For>
						</Show>
					</TableBody>
				</Table>
			</div>

			{/* Pagination */}
			<Show when={pageCount() > 1}>
				<div class="flex items-center justify-between px-1">
					<p class="text-xs text-faint">
						Showing {start()}–{end()} of {filteredCount()} jobs
					</p>
					<div class="flex items-center gap-2">
						<button
							type="button"
							onClick={() => table.previousPage()}
							disabled={!table.getCanPreviousPage()}
							class="inline-flex items-center gap-1 rounded-md border border-border bg-surface px-3 py-1.5 text-sm font-medium text-muted transition hover:border-border-strong hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
						>
							← Prev
						</button>
						<span class="text-sm text-faint">
							Page {pageIndex() + 1} of {pageCount()}
						</span>
						<button
							type="button"
							onClick={() => table.nextPage()}
							disabled={!table.getCanNextPage()}
							class="inline-flex items-center gap-1 rounded-md border border-border bg-surface px-3 py-1.5 text-sm font-medium text-muted transition hover:border-border-strong hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
						>
							Next →
						</button>
					</div>
				</div>
			</Show>
		</div>
	);
}
